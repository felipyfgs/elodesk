-- 0010_channels_telegram.sql: Telegram channel table + indexes

CREATE TABLE IF NOT EXISTS channels_telegram (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    bot_token_ciphertext TEXT NOT NULL,
    bot_name TEXT,
    webhook_identifier TEXT NOT NULL,
    secret_token_ciphertext TEXT NOT NULL,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_telegram_account_id ON channels_telegram(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_telegram_webhook_identifier
    ON channels_telegram(webhook_identifier);
