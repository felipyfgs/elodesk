-- 0008_channels_instagram.sql: Instagram channel table

CREATE TABLE IF NOT EXISTS channels_instagram (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    instagram_id TEXT NOT NULL,
    access_token_ciphertext TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '60 days'),
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_instagram_account_id ON channels_instagram(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_instagram_instagram_id
    ON channels_instagram(instagram_id);
