-- 0007_channels_email.sql: Email channel tables

-- Make inboxes.channel_id polymorphic (no DB-level FK) so both
-- channels_api and channels_email rows can be referenced.
ALTER TABLE inboxes DROP CONSTRAINT IF EXISTS inboxes_channel_id_fkey;

CREATE TABLE IF NOT EXISTS channels_email (
    id                       BIGSERIAL PRIMARY KEY,
    account_id               BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    email                    TEXT NOT NULL,
    name                     TEXT NOT NULL DEFAULT '',
    provider                 TEXT NOT NULL DEFAULT 'generic' CHECK (provider IN ('generic','google','microsoft')),

    -- IMAP
    imap_address             TEXT,
    imap_port                INTEGER,
    imap_login               TEXT,
    imap_password_ciphertext TEXT,
    imap_enable_ssl          BOOLEAN NOT NULL DEFAULT TRUE,
    imap_enabled             BOOLEAN NOT NULL DEFAULT FALSE,
    last_uid_seen            BIGINT NOT NULL DEFAULT 0,

    -- SMTP
    smtp_address             TEXT,
    smtp_port                INTEGER,
    smtp_login               TEXT,
    smtp_password_ciphertext TEXT,
    smtp_enable_ssl          BOOLEAN NOT NULL DEFAULT TRUE,

    -- OAuth provider_config stores { access_token, refresh_token, expires_on }
    -- as AES-GCM ciphertext JSON; NULL for generic IMAP/SMTP
    provider_config          TEXT,

    verified_for_sending     BOOLEAN NOT NULL DEFAULT FALSE,
    requires_reauth          BOOLEAN NOT NULL DEFAULT FALSE,

    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channels_email_account_id ON channels_email(account_id);

-- Transient OAuth state: state -> pending inbox creation.
-- Rows are short-lived (10 min TTL enforced in application).
CREATE TABLE IF NOT EXISTS email_oauth_pending (
    state      TEXT PRIMARY KEY,
    account_id BIGINT NOT NULL,
    inbox_name TEXT NOT NULL,
    provider   TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
