-- 0042_conversations_assignee_last_seen_at.sql: tracks when the assignee last
-- viewed the conversation, used to compute unread_count in the hydrated list
-- query. NULL means "never seen" — every incoming message counts as unread.

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS assignee_last_seen_at TIMESTAMPTZ;
