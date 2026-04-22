-- 0024_channels_tiktok.sql: TikTok Business Messaging channel table + indexes

CREATE TABLE IF NOT EXISTS channels_tiktok (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    business_id TEXT NOT NULL,
    access_token_ciphertext TEXT NOT NULL,
    refresh_token_ciphertext TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    refresh_token_expires_at TIMESTAMPTZ NOT NULL,
    display_name TEXT,
    username TEXT,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_tiktok_account_id ON channels_tiktok(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_tiktok_business_id
    ON channels_tiktok(business_id);
