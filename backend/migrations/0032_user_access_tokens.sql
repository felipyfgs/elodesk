-- Migration: user_access_tokens
-- Description: Polymorphic user access tokens for API authentication

CREATE TABLE IF NOT EXISTS user_access_tokens (
    id BIGSERIAL PRIMARY KEY,
    owner_type TEXT NOT NULL,
    owner_id BIGINT NOT NULL,
    token TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Unique index on token for fast lookup during authentication
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_access_tokens_token ON user_access_tokens(token);

-- Unique constraint on owner: each user can have at most one access token
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_access_tokens_owner ON user_access_tokens(owner_type, owner_id);

-- Backfill: create tokens for existing users who don't have one
-- This is done via application code in the auth service
