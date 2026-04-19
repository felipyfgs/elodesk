-- 0002_inbox_channel_api.sql: Inbox and Channel API tables

CREATE TABLE IF NOT EXISTS channels_api (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    webhook_url TEXT,
    identifier TEXT NOT NULL UNIQUE,
    hmac_token TEXT NOT NULL UNIQUE,
    hmac_mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    secret TEXT,
    api_token TEXT,
    additional_attributes JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channels_api_account_id ON channels_api(account_id);

CREATE TABLE IF NOT EXISTS inboxes (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    channel_id BIGINT NOT NULL REFERENCES channels_api(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    channel_type TEXT NOT NULL DEFAULT 'Channel::Api',
    enable_auto_assignment BOOLEAN DEFAULT TRUE,
    greeting_message TEXT,
    allow_messages_after_resolved BOOLEAN DEFAULT TRUE,
    lock_to_single_conversation BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inboxes_account_id ON inboxes(account_id);
CREATE INDEX idx_inboxes_channel ON inboxes(channel_id, channel_type);
CREATE INDEX idx_inboxes_account_id_id ON inboxes(account_id, id);
