ALTER TABLE messages ADD COLUMN forwarded_from_message_id BIGINT NULL REFERENCES messages(id) ON DELETE SET NULL;

CREATE INDEX idx_messages_forwarded_from ON messages(forwarded_from_message_id) WHERE forwarded_from_message_id IS NOT NULL;
