-- 0026_channels_twitter.sql: Twitter/X DM channel table + indexes

CREATE TABLE IF NOT EXISTS channels_twitter (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    profile_id TEXT NOT NULL,
    screen_name TEXT,
    twitter_access_token_ciphertext TEXT NOT NULL,
    twitter_access_token_secret_ciphertext TEXT NOT NULL,
    tweets_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_twitter_account_id ON channels_twitter(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_twitter_account_profile
    ON channels_twitter(account_id, profile_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_twitter_profile_id
    ON channels_twitter(profile_id);
