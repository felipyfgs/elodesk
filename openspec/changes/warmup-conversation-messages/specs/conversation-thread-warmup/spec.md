## ADDED Requirements

### Requirement: Message bucket warming from conversation list payload

The frontend message store SHALL seed a per-conversation message bucket with the conversation's `lastNonActivityMessage` whenever the conversation is hydrated into the conversations store and the bucket for that conversation is empty.

The seed SHALL be inserted using the same upsert semantics used for full fetches, so subsequent realtime arrivals or full history fetches deduplicate against the seed by message id.

#### Scenario: Hydrating the conversation list seeds empty buckets

- **WHEN** the agent loads the dashboard and the conversations store receives a list response that includes a conversation `C` with `lastNonActivityMessage = M`
- **AND** the message store has no bucket for `C` (or the bucket is empty)
- **THEN** the message store contains exactly one entry for `C`, equal to `M`
- **AND** the per-conversation fetch state for `C` is `warmed`

#### Scenario: Hydrating a conversation with no last message leaves the bucket empty

- **WHEN** the conversations store receives a conversation `C` with `lastNonActivityMessage = null`
- **THEN** the message store has no bucket entry created for `C`
- **AND** the per-conversation fetch state for `C` is `empty` (or absent)

#### Scenario: Re-hydrating an already-fetched conversation does not overwrite history

- **WHEN** the message store has a `fetched` bucket for conversation `C` containing messages `[M1, M2, M3]`
- **AND** the conversations store re-hydrates `C` with a fresh `lastNonActivityMessage = M3`
- **THEN** the bucket for `C` remains `[M1, M2, M3]` unchanged
- **AND** the per-conversation fetch state for `C` remains `fetched`

#### Scenario: Warmup applies to deep-link single-conversation fetches

- **WHEN** the agent navigates directly to `/conversations/:id` for a conversation `C` not present in the cached list
- **AND** the dashboard fetches `C` individually and that response carries `lastNonActivityMessage = M`
- **THEN** the message bucket for `C` contains `M` before the thread component mounts
- **AND** the per-conversation fetch state for `C` is `warmed`

#### Scenario: Warmup applies when realtime delivers a previously unknown conversation

- **WHEN** the realtime channel delivers `conversation.created` for a conversation `C` with `lastNonActivityMessage = M`
- **AND** the message store has no bucket for `C`
- **THEN** the message store contains an entry for `C` equal to `M`
- **AND** the per-conversation fetch state for `C` is `warmed`

### Requirement: Per-conversation fetch state tracks warmup separately from full fetch

The frontend message store SHALL maintain a fetch state per conversation with the values `empty`, `warmed`, `fetching`, and `fetched`, so that prefetch and mount-time fetch logic distinguish a seed-only bucket from a fully fetched one.

#### Scenario: Prefetch runs on a warmed bucket

- **WHEN** the agent hovers a conversation row whose bucket is in state `warmed`
- **THEN** the message store dispatches a full history fetch for that conversation
- **AND** the per-conversation fetch state transitions to `fetching`

#### Scenario: Prefetch is a no-op on a fetched bucket within TTL

- **WHEN** the agent hovers a conversation row whose bucket is in state `fetched`
- **AND** the most recent successful fetch for that conversation is within the freshness TTL
- **THEN** the message store does not issue a new request
- **AND** the per-conversation fetch state remains `fetched`

#### Scenario: Concurrent prefetch and click do not duplicate requests

- **WHEN** a hover prefetch is in flight for conversation `C` (`fetching`)
- **AND** the agent clicks `C` causing the thread to mount and call `fetchMessages`
- **THEN** only one HTTP request is in flight for `C`
- **AND** both call sites observe the same eventual result

#### Scenario: Successful full fetch transitions state to fetched

- **WHEN** a full history fetch for conversation `C` completes successfully
- **THEN** the per-conversation fetch state for `C` is `fetched`
- **AND** the last-fetched timestamp for `C` is updated

### Requirement: Conversation upsert preserves the message bucket

The conversations store SHALL NOT mutate or replace the per-conversation message bucket owned by the message store when handling `setAll`, `upsert`, or realtime conversation updates.

#### Scenario: Realtime conversation update keeps existing messages

- **WHEN** the message store has a bucket for conversation `C` containing `[M1, M2, M3]` (state `fetched`)
- **AND** the realtime channel delivers `conversation.updated` for `C` with new attributes (status change, assignee change, etc.)
- **THEN** the bucket for `C` remains `[M1, M2, M3]` after the update
- **AND** the per-conversation fetch state for `C` remains `fetched`

#### Scenario: Realtime conversation update warms an empty bucket

- **WHEN** the message store has no bucket for conversation `C`
- **AND** the realtime channel delivers `conversation.updated` for `C` with `lastNonActivityMessage = M`
- **THEN** the message store creates a bucket for `C` containing exactly `M`
- **AND** the per-conversation fetch state for `C` is `warmed`

### Requirement: Prefetch coverage extends to keyboard navigation and deep-link

The conversations list SHALL trigger a message prefetch when the agent navigates with the keyboard (`ArrowDown`, `ArrowUp`, `Enter`, `Space`), in addition to mouse `mouseenter` and `focus`. Deep-link navigation that ends with `ensureSelectedLoaded` SHALL also trigger a prefetch as soon as the conversation id is known.

#### Scenario: Arrow key navigation prefetches the next selected conversation

- **WHEN** the agent presses `ArrowDown` while a conversation is selected and there is a next conversation in the list
- **THEN** a message prefetch is dispatched for the next conversation before the selection moves
- **AND** the prefetch follows the same dedup rules as hover prefetch

#### Scenario: Deep-link triggers prefetch after deep-fetching the conversation

- **WHEN** the agent loads a URL of the form `/conversations/:id` for a conversation absent from the cached list
- **AND** `ensureSelectedLoaded` fetches the conversation and stores it
- **THEN** a message prefetch is dispatched for that conversation immediately after the conversation is added to the store
- **AND** the prefetch obeys the same fetch state transitions as the hover prefetch

### Requirement: Thread mount-time fetch coexists with warmup

The conversation thread component SHALL still dispatch a full message fetch on mount (or when the selected conversation id changes), regardless of the bucket's current fetch state, unless the bucket is already `fetched` within the freshness TTL.

The merge of fetched messages with the bucket SHALL preserve any messages already present (warmup seed, optimistic placeholders, realtime arrivals) by id-based upsert.

#### Scenario: Thread mount with a warmed bucket fills history above the seed

- **WHEN** the thread mounts for conversation `C` whose bucket is `[M5]` (warmup seed) in state `warmed`
- **AND** the full history fetch returns `[M1, M2, M3, M4, M5]`
- **THEN** the bucket for `C` is `[M1, M2, M3, M4, M5]` after the merge
- **AND** the per-conversation fetch state for `C` is `fetched`
- **AND** no message id is duplicated

#### Scenario: Thread mount with an empty bucket renders empty until fetch completes

- **WHEN** the thread mounts for conversation `C` whose bucket is empty (state `empty`)
- **AND** `C` had `lastNonActivityMessage = null` at hydration time (e.g. brand-new conversation with no messages)
- **THEN** the thread renders the empty state until the fetch completes
- **AND** no warmup seed is fabricated

#### Scenario: Realtime message arriving during fetch is preserved

- **WHEN** the thread for conversation `C` is in state `fetching` and the bucket is `[M5]` (warmup seed)
- **AND** the realtime channel delivers `message.created` for a new message `M6` belonging to `C`
- **AND** the in-flight fetch then returns `[M1..M5]`
- **THEN** the final bucket is `[M1, M2, M3, M4, M5, M6]`
- **AND** no message is dropped

### Requirement: Warmup preserves message-type semantics already enforced by the backend

The warmup seed SHALL inherit whatever visibility, privacy, and message-type filters the backend applied when computing `lastNonActivityMessage`. The frontend SHALL NOT introduce additional filtering at warmup time.

#### Scenario: Private (internal note) seed is rendered like any other private message

- **WHEN** a conversation `C` is hydrated with `lastNonActivityMessage = M` where `M.private = true`
- **THEN** the warmup seed for `C` includes `M` exactly as delivered
- **AND** the thread renders `M` using the existing private-note styling once the thread is opened
