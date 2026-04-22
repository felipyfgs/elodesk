-- 0027_channels_wzap.sql: wzap (whatsmeow-based) native channel table

CREATE TABLE IF NOT EXISTS channels_wzap (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    base_url TEXT NOT NULL,
    admin_token_ciphertext TEXT NOT NULL,
    session_id TEXT NOT NULL,
    webhook_secret_ciphertext TEXT NOT NULL,
    outbound_webhook_url TEXT,
    engine_phone TEXT,
    qr_state TEXT NOT NULL DEFAULT 'none',
    connection_state TEXT NOT NULL DEFAULT 'disconnected',
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_wzap_session_id ON channels_wzap(session_id);
CREATE INDEX IF NOT EXISTS idx_channels_wzap_account_id ON channels_wzap(account_id);
