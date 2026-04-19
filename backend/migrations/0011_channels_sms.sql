-- 0010_channels_sms.sql: SMS channel table + phone_e164 on contacts

CREATE TABLE IF NOT EXISTS channels_sms (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    inbox_id BIGINT REFERENCES inboxes(id) ON DELETE SET NULL,
    provider TEXT NOT NULL CHECK (provider IN ('twilio', 'bandwidth', 'zenvia')),
    phone_number TEXT NOT NULL,
    webhook_identifier TEXT NOT NULL,
    provider_config_ciphertext TEXT NOT NULL DEFAULT '',
    messaging_service_sid TEXT,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_sms_account_id ON channels_sms(account_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_sms_webhook_identifier ON channels_sms(webhook_identifier);
CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_sms_account_phone ON channels_sms(account_id, phone_number);
CREATE INDEX IF NOT EXISTS idx_channels_sms_inbox_id ON channels_sms(inbox_id) WHERE inbox_id IS NOT NULL;

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS phone_e164 TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_phone_e164_account
    ON contacts(phone_e164, account_id)
    WHERE phone_e164 IS NOT NULL AND phone_e164 != '';
