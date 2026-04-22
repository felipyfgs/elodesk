-- 0023_channels_line.sql: LINE Messaging API channel table + indexes

CREATE TABLE IF NOT EXISTS channels_line (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    line_channel_id TEXT NOT NULL,
    line_channel_secret_ciphertext TEXT NOT NULL,
    line_channel_token_ciphertext TEXT NOT NULL,
    bot_basic_id TEXT,
    bot_display_name TEXT,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_line_account_id ON channels_line(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_line_line_channel_id
    ON channels_line(line_channel_id);
