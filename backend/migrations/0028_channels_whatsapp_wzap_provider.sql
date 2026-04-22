-- 0028_channels_whatsapp_wzap_provider.sql: store wzap provider config on Channel::Whatsapp

ALTER TABLE channels_whatsapp
    ADD COLUMN IF NOT EXISTS base_url TEXT,
    ADD COLUMN IF NOT EXISTS session_id TEXT,
    ADD COLUMN IF NOT EXISTS webhook_secret_ciphertext TEXT,
    ADD COLUMN IF NOT EXISTS outbound_webhook_url TEXT,
    ADD COLUMN IF NOT EXISTS engine_phone TEXT,
    ADD COLUMN IF NOT EXISTS qr_state TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS connection_state TEXT NOT NULL DEFAULT 'disconnected',
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_whatsapp_wzap_session_id
    ON channels_whatsapp(session_id)
    WHERE provider = 'wzap' AND session_id IS NOT NULL AND session_id != '';
