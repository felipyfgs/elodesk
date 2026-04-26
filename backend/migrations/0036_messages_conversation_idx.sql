-- 0036_messages_conversation_idx.sql: composite index supporting the LATERAL
-- subquery in conversation_repo's hydrated list — pulls last non-activity
-- message per conversation in O(log N).

CREATE INDEX IF NOT EXISTS messages_conv_msgtype_created_idx
    ON messages (conversation_id, message_type, created_at DESC);
