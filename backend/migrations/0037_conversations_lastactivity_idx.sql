-- 0037_conversations_lastactivity_idx.sql: partial index for the default
-- conversation listing (account scope, ordered by last_activity_at DESC,
-- excluding resolved). Status enum: Open=0, Resolved=1, Pending=2, Snoozed=3.

CREATE INDEX IF NOT EXISTS conversations_account_lastactivity_idx
    ON conversations (account_id, last_activity_at DESC)
    WHERE status != 1;
