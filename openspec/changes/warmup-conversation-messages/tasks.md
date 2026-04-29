## 1. Message store: per-conversation fetch state

- [x] 1.1 Add `fetchState: Record<string, 'empty' | 'warmed' | 'fetching' | 'fetched'>` to `useMessagesStore` state in `frontend/app/stores/messages.ts`, alongside existing `byConversation` / `fetchedAt` / `inflight`.
- [x] 1.2 Replace internal reads of `inflight: Set<string>` with the `fetchState === 'fetching'` check; remove the `inflight` field once no callers remain.
- [x] 1.3 Update `set(conversationId, list)` to also write `fetchState[conversationId] = 'fetched'` and `fetchedAt[conversationId] = Date.now()`.
- [x] 1.4 Update `mergeFetched` to set `fetchState[conversationId] = 'fetched'` after a successful merge (do not regress `warmed` → `fetched` on partial merges that only re-insert the seed).
- [x] 1.5 Update `fetchMessages` to transition state: `empty|warmed → fetching` on entry, `fetching → fetched` on success, `fetching → previous` on error. Keep the freshness TTL guard for `fetched` entries only.

## 2. Message store: warmup action

- [x] 2.1 Add `warmIfEmpty(conversation)` action to `useMessagesStore` that seeds `byConversation[c.id]` with `c.lastNonActivityMessage` and sets `fetchState[c.id] = 'warmed'` only when `byConversation[c.id]` is empty/undefined and `c.lastNonActivityMessage` is present.
- [x] 2.2 Make `warmIfEmpty` a no-op when `fetchState[c.id]` is `fetching` or `fetched`.
- [x] 2.3 Update `prefetch(conversationId)` to dispatch `fetchMessages` for both `empty` and `warmed` states (current heuristic only checks `length > 0` and incorrectly skips warmed buckets).

## 3. Conversations store: warm on hydration

- [x] 3.1 In `frontend/app/stores/conversations.ts`, after `setAll(list)` writes the new list, iterate `list` and call `useMessagesStore().warmIfEmpty(c)` for each conversation.
- [x] 3.2 In `upsert(c)`, after writing the conversation record, call `useMessagesStore().warmIfEmpty(c)`.
- [x] 3.3 Add an inline comment in `upsert` documenting that messages are owned by `useMessagesStore` and must not be touched here, to lock the invariant for future refactors.

## 4. Deep-link path warms before mount

- [x] 4.1 In `frontend/app/components/conversations/Index.vue`, inside `ensureSelectedLoaded`, after `convs.upsert(conv)` and `convs.setCurrent(conv)`, call `useMessagesStore().prefetch(conv.id)` so the deep-fetched conversation gets a prefetch in addition to the warmup that `upsert` already triggered.
- [x] 4.2 Verify that the existing `Thread.vue` watcher (`watch(() => props.conversation.id, ...)`) still fires `messages.fetchMessages` on mount; the store dedup makes this idempotent with the prefetch above.

## 5. Realtime: warm on conversation events

- [x] 5.1 In `frontend/app/composables/useConversationRealtime.ts`, the existing `conversation.created` and `conversation.updated` handlers already call `convs.upsert(c)` — no extra wiring needed once task 3.2 lands. Confirm with a manual trace and add a comment in the handler pointing at the warmup invariant.
- [x] 5.2 Confirm the `message.created` handler does not need warmup changes (it already calls `messages.upsert`); add an assertion-style comment if helpful.

## 6. Keyboard prefetch in the list

- [x] 6.1 In `frontend/app/components/conversations/List.vue`, modify the `arrowdown` handler in `defineShortcuts` to compute the next conversation, call `messages.prefetch(next.id)`, then assign it to `selected`.
- [x] 6.2 Mirror the change in the `arrowup` handler.
- [x] 6.3 Update the row's `@keydown.enter.prevent` and `@keydown.space.prevent` handlers to also call `prefetchMessages(c)` before `selectConversation(c)` (covers the case where the agent tabs into a row and presses Enter without ever hovering).

## 7. Type updates

- [x] 7.1 Export the `FetchState` type from `frontend/app/stores/messages.ts` so test files and any future consumers can reference the discriminated union by name.
- [x] 7.2 If `Conversation` interface in `frontend/app/stores/conversations.ts` doesn't already type `lastNonActivityMessage` strongly enough for the store action, tighten it.

## 8. Tests

- [ ] 8.1 Add unit tests for `useMessagesStore.warmIfEmpty`: empty bucket gets seeded; non-empty bucket is preserved; null `lastNonActivityMessage` is a no-op; `fetching`/`fetched` states are not regressed.
- [ ] 8.2 Add unit tests for `useMessagesStore.prefetch` with the new states: triggers fetch on `empty`/`warmed`; skips on `fetching`/`fetched` within TTL; concurrent calls collapse into one in-flight request.
- [ ] 8.3 Add a test for `mergeFetched` covering the warmup case: bucket starts as `[seed]`, fetch returns `[m1..m5]` including the seed, final bucket has no duplicates and is correctly ordered.
- [ ] 8.4 Add a test for the realtime race: bucket is `[seed]`, realtime delivers a newer message before the fetch completes, fetch returns historical messages — final bucket contains both historical messages and the realtime message.
- [ ] 8.5 Add a test for `useConversationsStore.upsert` confirming the message bucket is not touched when an existing conversation is updated.
- [ ] 8.6 Add a test for the deep-link path that exercises `ensureSelectedLoaded` with a stub API and asserts the bucket is warmed and prefetched.

## 9. Verification

- [x] 9.1 Run `pnpm typecheck` from `frontend/` and resolve any new type errors.
- [x] 9.2 Run `pnpm lint` from `frontend/`.
- [ ] 9.3 Manual smoke: cold-load `/conversations`, click a conversation without hovering — confirm the last message renders before any network response.
- [ ] 9.4 Manual smoke: keyboard nav with `↓`/`↑`, confirm the next thread renders the seed instantly.
- [ ] 9.5 Manual smoke: deep-link `/conversations/:id` with a freshly loaded tab — confirm the seed renders before history arrives.
- [ ] 9.6 Manual smoke: long conversation with hundreds of messages — confirm the seed appears at the bottom while the rest of the history streams in above without scroll jumps.
