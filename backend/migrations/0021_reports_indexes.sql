-- 0021_reports_indexes.sql: indexes to keep reports aggregations fast.

CREATE INDEX IF NOT EXISTS idx_conversations_account_status_created
    ON conversations (account_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversations_assignee_created
    ON conversations (assignee_id, created_at DESC)
    WHERE assignee_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_conversation_created
    ON messages (conversation_id, created_at DESC);
