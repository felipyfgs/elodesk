-- 0033_accounts_settings.sql: Extend accounts table with locale, status, settings, custom_attributes

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS locale            TEXT     NOT NULL DEFAULT 'pt',
    ADD COLUMN IF NOT EXISTS status            SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS custom_attributes JSONB    NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS settings          JSONB    NOT NULL DEFAULT '{}';

-- Index to quickly filter active/suspended accounts
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts (status);
