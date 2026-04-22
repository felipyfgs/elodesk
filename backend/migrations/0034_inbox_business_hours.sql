-- 0034_inbox_business_hours.sql: per-inbox business hours configuration

CREATE TABLE IF NOT EXISTS inbox_business_hours (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    inbox_id BIGINT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    schedule JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (inbox_id)
);

CREATE INDEX IF NOT EXISTS idx_inbox_business_hours_account
    ON inbox_business_hours(account_id);
