-- 0039_participants.sql: group conversation membership. One row per
-- (conversation, contact) pair; populated by Wzap when syncing WhatsApp
-- group members.

CREATE TABLE IF NOT EXISTS participants (
    id              BIGSERIAL PRIMARY KEY,
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    contact_id      BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    role            TEXT NOT NULL DEFAULT 'member',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, contact_id)
);

CREATE INDEX IF NOT EXISTS participants_conv_idx ON participants(conversation_id);
CREATE INDEX IF NOT EXISTS participants_account_idx ON participants(account_id);
