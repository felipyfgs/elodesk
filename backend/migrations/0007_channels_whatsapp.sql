-- 0006_channels_whatsapp.sql: WhatsApp channel table + indexes

CREATE TABLE IF NOT EXISTS channels_whatsapp (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT 'whatsapp_cloud',
    phone_number TEXT NOT NULL DEFAULT '',
    phone_number_id TEXT,
    business_account_id TEXT,
    api_key_ciphertext TEXT NOT NULL,
    webhook_verify_token_ciphertext TEXT,
    message_templates JSONB NOT NULL DEFAULT '[]',
    message_templates_synced_at TIMESTAMPTZ,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_whatsapp_account_id ON channels_whatsapp(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_whatsapp_account_phone
    ON channels_whatsapp(account_id, phone_number)
    WHERE phone_number IS NOT NULL AND phone_number != '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_whatsapp_phone_number_id
    ON channels_whatsapp(phone_number_id)
    WHERE phone_number_id IS NOT NULL;

ALTER TABLE inboxes DROP CONSTRAINT IF EXISTS inboxes_channel_id_fkey;
ALTER TABLE inboxes ADD CONSTRAINT inboxes_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels_api(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_inboxes_channel_type ON inboxes(channel_type);
