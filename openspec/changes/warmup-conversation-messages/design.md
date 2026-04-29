## Context

Today the Vue dashboard discards a piece of data the backend already pays to compute. Every conversation in `GET /accounts/:aid/conversations` carries the most recent non-activity message in two equivalent fields:

```
ConversationResp {
  ...
  "messages": [<lastNonActivityMessage>],         // Chatwoot-shape, single-element array
  "last_non_activity_message": {<same message>}   // explicit field used by our adapter
}
```

The list view (`@/home/obsidian/dev/project/elodesk/frontend/app/components/conversations/List.vue:80-99`) reads this for the row preview. The thread view (`@/home/obsidian/dev/project/elodesk/frontend/app/components/conversations/Thread.vue:65-71`) ignores it: on every selection it calls `messages.fetchMessages(id)` with an empty bucket, waits for the REST round-trip, then renders. The hover-prefetch in `@/home/obsidian/dev/project/elodesk/frontend/app/components/conversations/List.vue:202-204` partially mitigates this, but only when the agent's pointer crosses the row long enough for `mouseenter` to fire and the response to arrive before the click.

The agent therefore experiences "loading" in four scenarios:

```
1. mouseenter ──► prefetch ──► (200ms) ──► click ──► instant       OK
2. mouseenter+click together  ──► fetch in flight ──► flash         BAD
3. ↓/↑ keyboard nav           ──► no prefetch ──► flash             BAD
4. /conversations/:id reload  ──► waterfall ──► flash               BAD
```

In Chatwoot the same setup is solved differently: `setActiveChat({ data })` (`@/home/obsidian/dev/project/_refs/chatwoot/app/javascript/dashboard/store/modules/conversations/actions.js:193-208`) accepts the conversation row directly from the list and treats `data.messages[0]` as the seed of the thread. The thread renders the seed immediately and dispatches `fetchPreviousMessages` with `before: data.messages[0].id` to fill history above it. The mutation `SET_ALL_CONVERSATION` (`@/home/obsidian/dev/project/_refs/chatwoot/app/javascript/dashboard/store/modules/conversations/index.js:48-58`) preserves the existing `messages` array on subsequent realtime updates so the seed and any later additions are never thrown away.

This change applies the same pattern to Elodesk while keeping our existing hover prefetch on top, which Chatwoot does not have.

## Goals / Non-Goals

**Goals:**
- Make the message thread render with at least one real message instantly when a conversation is opened, regardless of how it was opened (mouse, keyboard, deep-link).
- Reuse data already on the wire (`lastNonActivityMessage`) instead of issuing extra requests at hydration time.
- Keep the hover-prefetch behavior intact for the cases where it already wins (full history ready before the click).
- Keep the existing realtime reconciliation correct: warmed seeds and full fetches must coexist with `message.created`/`message.updated`/`message.deleted` arrivals without losing or duplicating messages.

**Non-Goals:**
- Backend changes. The DTO already exposes the seed; we do not introduce `before=<id>` pagination or any new query param in this change.
- A "prefetch top-N conversations on idle" strategy. Discussed during exploration; postponed until we have telemetry showing the warmup is insufficient.
- Persisting message buckets to `localStorage` or any disk cache.
- Replacing the existing fetch + merge logic. The full-history fetch on mount remains; we change *what the bucket already contains* when that fetch starts.

## Decisions

### D1. Warm the bucket from `lastNonActivityMessage`, not `messages[0]`

The store today already normalises through `apiAdapter`, and `Conversation.lastNonActivityMessage` is the canonical adapted shape used by the list preview. The Chatwoot-shape `messages[0]` is the same data point but goes through a different normalisation path. Using `lastNonActivityMessage` keeps the warmup aligned with the rest of the frontend and makes type assertions trivial.

**Alternatives considered:**
- *Use `c.messages[0]`*: closer to Chatwoot, but in our DTO `messages` and `last_non_activity_message` are the same source — `messages` is just `[lastNonActivityMessage]`. No advantage, slightly more friction with our adapter.
- *Store an explicit `seed` field on the conversation record and hydrate the thread reactively*: clean but requires a new field on the conversation type and complicates the WS upsert path. The bucket-seeding approach reuses every existing rendering path in `Thread.vue` / `MessageList.vue` with no template changes.

### D2. Track per-conversation fetch state with a `fetchState` map

The current `prefetch()` short-circuits on `byConversation[id].length > 0`. Once we seed the bucket with one message, that heuristic incorrectly returns "already cached" for buckets that only have the warmup seed. Chatwoot solves this with a per-conversation `dataFetched` flag stored on the conversation object itself.

We introduce a parallel state in `useMessagesStore`:

```ts
fetchState: Record<string, 'empty' | 'warmed' | 'fetching' | 'fetched'>
```

- `empty` – no entry; default
- `warmed` – seed-only; `prefetch()` and the Thread mount fetch must still run
- `fetching` – request in flight (replaces today's `inflight: Set`)
- `fetched` – full history fetched at least once; respect `freshMs` TTL before refetching

`fetchedAt` is already a `Record<string, number>` and stays as the TTL bookkeeping for `fetched` entries.

**Alternatives considered:**
- *Augment the `Conversation` type with `dataFetched` like Chatwoot*: works, but the Chatwoot store keeps messages on the conversation object itself; ours splits them across two stores. Putting the state next to the data it describes (the bucket) keeps it inside `useMessagesStore`.
- *Boolean `fetched: Record<string, boolean>`*: insufficient — we need to differentiate "seed only" from "empty" so realtime arrivals can decide whether the bucket already had a fully-fetched history.

### D3. Preserve the bucket on `convs.upsert`, mirror Chatwoot's `SET_ALL_CONVERSATION`

`useConversationsStore.upsert` is invoked by realtime (`conversation.created`/`conversation.updated`) and by `convs.setAll`. The new conversation record carries an updated `lastNonActivityMessage`, but the message bucket lives in `useMessagesStore` already — the bucket isn't on the conversation object, so `upsert` already cannot blow it away. We still need to defend against accidental coupling: the warmup seed is now functionally part of the bucket, and a future refactor that stuffed `messages` onto the conversation could regress this. We will:

1. Document explicitly in `useConversationsStore.upsert` that messages are owned by `useMessagesStore` and never touched here.
2. After upserting a conversation, call `messages.warmIfEmpty(c)` so newly arrived conversations that haven't been opened yet get a seed for their next render.

The realtime `message.created` handler in `useConversationRealtime` already calls `messages.upsert(m)` and then `applyConversationSummary` — that path is untouched and continues to dedupe via `upsert`'s id-based reconciliation.

### D4. Extend prefetch to keyboard, focus, and deep-link

`prefetchMessages(c)` runs today on `mouseenter` and `focus`. The keyboard shortcut handler in `defineShortcuts` (`@/home/obsidian/dev/project/elodesk/frontend/app/components/conversations/List.vue:151-162`) only mutates `selected.value`. We will:

1. Call `messages.prefetch(target.id)` in the `arrowdown`/`arrowup` shortcut handlers right before mutating `selected.value`.
2. Call `messages.prefetch(c.id)` from `ensureSelectedLoaded` in `ConversationsIndex.vue` after the deep-fetched conversation is known, so deep-links and reloads start the warmup + history fetch as soon as we have an id.
3. Keep `mouseenter` and `focus` as-is. With `fetchState`, repeated calls become cheap idempotent no-ops once `fetched`/`fetching`.

### D5. `Thread.vue`'s mount-time fetch stays, but coexists with warmup

`watch(() => props.conversation.id, async (id) => { await messages.fetchMessages(id); markRead(id) })` continues to run with `immediate: true`. The merge logic in `messages.mergeFetched` already preserves messages already in the bucket — the seed survives because `upsert` is idempotent on id match. The fetch updates `fetchState` from `warmed`/`empty` → `fetching` → `fetched`. The visible result is: one bubble appears the moment the Thread mounts, the rest stream in from the REST response (typical 100–300 ms later) and prepend in chronological order.

Empty conversations (no `lastNonActivityMessage`) behave exactly as today: bucket stays empty, fetch runs, thread renders when the response lands. No regression for fresh conversations with zero messages.

### D6. Privacy of warmed seed

`last_non_activity_message` may include private notes if the most recent non-activity message is one. The list preview already shows them with the lock icon (`@/home/obsidian/dev/project/elodesk/frontend/app/components/conversations/List.vue:88-89`). The warmed bucket therefore mirrors what the agent can already see in the list — no new exposure surface. We add an explicit acceptance test for this so the contract doesn't drift.

## Risks / Trade-offs

- **[Stale seed when WS missed an update]** → If the agent loaded the list, lost WS, and reconnected after a new message, the seed could be older than what's on the server. *Mitigation*: the mount-time fetch (`Thread.vue`'s `watch`) still runs and merges — the bucket converges within one round-trip. Worst case is a brief moment showing the previous last message before history catches up; identical to today's behaviour minus the empty flash.

- **[One bubble + empty area looks broken]** → On a tall desktop viewport with `justify-end`, a single bubble pinned to the bottom is the natural state of a one-message conversation, so it reads as "loaded". On a conversation with hundreds of messages, the agent sees one bubble appear, then the history fills above it. *Mitigation*: this is the same UX Chatwoot ships and matches WhatsApp Web's "tail-first" loading; we accept it. If user testing surfaces complaints we revisit with a small "loading earlier messages…" indicator at the top.

- **[`fetchState` desync with `byConversation`]** → If a future writer mutates `byConversation` directly without updating `fetchState`, prefetch heuristics could regress. *Mitigation*: keep all writes funneled through store actions (`warmIfEmpty`, `mergeFetched`, `upsert`, `set`); the `set` action becomes a thin wrapper that also writes `fetchState='fetched'`. Add a unit test asserting the invariants.

- **[`prefetch()` heuristic change ripples elsewhere]** → If any consumer relied on the old `length > 0` short-circuit (e.g. assumed a non-empty bucket meant "fully fetched"), the new states make that assumption invalid. *Mitigation*: the only consumer is `prefetchMessages` in `List.vue`. We migrate it explicitly and grep the codebase for direct reads of `byConversation` length to confirm no external assumption exists.

- **[Realtime arrival between warmup and full fetch]** → If `message.created` arrives after warmup but before the REST response, the bucket has `[seed, newMsg]`; when REST returns it includes seed but not the newer one — `mergeFetched` upserts the older messages and leaves `newMsg` in place. *Mitigation*: this is already the expected behaviour of `mergeFetched` and is covered by today's WhatsApp/SMS test scenarios. New unit test covers the warmup variant explicitly.

- **[Deep-link to a conversation absent from the list]** → `ensureSelectedLoaded` fetches the conversation single. The response has `lastNonActivityMessage`, so we warm before `Thread.vue` mounts. *Mitigation*: warmup must be invoked from `ensureSelectedLoaded` (D4 step 2), not just from `setAll`/`upsert`. Add an integration-style test that covers this path with a fake API.
