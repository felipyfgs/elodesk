## Why

Agents perceive a noticeable "loading" flash when opening a conversation, even though the backend already returns the last message of every conversation in the list payload (`last_non_activity_message` / `messages[0]`). The frontend currently uses that data only for the list preview and ignores it on the thread side, so opening a conversation always starts from an empty bucket and waits for a fresh REST round-trip. Combined with gaps in the existing hover-prefetch (no prefetch on keyboard navigation, deep-links, or fast clicks), this defeats the goal of the conversation feeling instantly available.

This change closes the gap by warming the message bucket from the data already in memory and tightening the prefetch coverage, so the thread renders with content the moment it is selected — matching the pattern Chatwoot uses (`setActiveChat` consumes the embedded `data.messages[0]` directly) and complementing our existing Linear-style hover prefetch.

## What Changes

- Seed `messages.byConversation[conversationId]` with the conversation's `lastNonActivityMessage` whenever a conversation is hydrated into `useConversationsStore` (initial list load, deep-link single fetch, realtime upsert) and the bucket is empty for that conversation.
- Distinguish "warmed-only" buckets from "fully fetched" buckets in `useMessagesStore` so that `prefetch()` and `Thread.vue`'s mount fetch still load full history when only the warmup seed is present.
- Preserve the existing `byConversation[id]` array when a conversation is updated through `useConversationsStore.upsert` (e.g. realtime), mirroring Chatwoot's `SET_ALL_CONVERSATION` mutation that replaces the conversation fields but keeps `messages`.
- Trigger `messages.prefetch(c.id)` on keyboard navigation (`arrowdown`/`arrowup`/`enter`/`space`) in `ConversationsList.vue`, closing the keyboard gap without changing existing mouse behaviour.
- Trigger `messages.prefetch(c.id)` from `ensureSelectedLoaded` in `ConversationsIndex.vue` so deep-links and reloads benefit from the warmup + prefetch even when the conversation isn't in the cached list yet.
- No backend changes. The DTO already exposes `last_non_activity_message`; this change only stops the frontend from discarding it.

## Capabilities

### New Capabilities
- `conversation-thread-warmup`: Hydration and pre-loading rules for the per-conversation message bucket on the agent dashboard, covering when buckets are warmed from list payloads, when prefetches fire, and how warmup interacts with the full-history fetch and realtime updates.

### Modified Capabilities
<!-- None: openspec/specs/ has no existing capability covering message hydration today. -->

## Impact

- **Frontend stores**: `frontend/app/stores/messages.ts` (new warmup action + per-conversation fetch state), `frontend/app/stores/conversations.ts` (call warmup on `setAll`/`upsert`, preserve message bucket on conversation upserts).
- **Frontend components**:
  - `frontend/app/components/conversations/List.vue` — extend prefetch trigger to keyboard handlers in `defineShortcuts`.
  - `frontend/app/components/conversations/Index.vue` — call prefetch from `ensureSelectedLoaded`.
  - `frontend/app/components/conversations/Thread.vue` — adjust mount-time fetch to coexist with warmup (no behaviour regression for buckets with only the seed).
- **Backend**: none. `last_non_activity_message` and the `messages` array are already produced by `dto.ConversationToRespFull` and the `/conversations` endpoints.
- **Realtime**: no protocol changes; existing `conversation.created` / `conversation.updated` payloads already carry `last_non_activity_message`, so realtime arrivals can warm the bucket using the same code path as REST hydration.
- **Tests**: new unit coverage for `useMessagesStore` warmup semantics and for the conversation upsert path that must preserve the bucket.
