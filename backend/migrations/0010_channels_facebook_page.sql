-- 0009_channels_facebook_page.sql: Facebook Page channel table

CREATE TABLE IF NOT EXISTS channels_facebook_page (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    page_id TEXT NOT NULL,
    page_access_token_ciphertext TEXT NOT NULL,
    user_access_token_ciphertext TEXT,
    instagram_id TEXT,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_facebook_page_account_id ON channels_facebook_page(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_facebook_page_page_id
    ON channels_facebook_page(page_id);

CREATE INDEX IF NOT EXISTS idx_channels_facebook_page_instagram_id
    ON channels_facebook_page(instagram_id)
    WHERE instagram_id IS NOT NULL;
