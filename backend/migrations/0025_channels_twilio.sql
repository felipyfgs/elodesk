-- 0025_channels_twilio.sql: Twilio dual-medium (sms|whatsapp) channel table

CREATE TABLE IF NOT EXISTS channels_twilio (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    medium TEXT NOT NULL CHECK (medium IN ('sms', 'whatsapp')),
    account_sid TEXT NOT NULL,
    auth_token_ciphertext TEXT NOT NULL,
    api_key_sid TEXT,
    phone_number TEXT,
    messaging_service_sid TEXT,
    content_templates JSONB NOT NULL DEFAULT '[]',
    content_templates_last_updated TIMESTAMPTZ,
    webhook_identifier TEXT NOT NULL,
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT channels_twilio_sender_xor CHECK (
        (phone_number IS NOT NULL AND messaging_service_sid IS NULL)
        OR (phone_number IS NULL AND messaging_service_sid IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_channels_twilio_account_id ON channels_twilio(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_twilio_webhook_identifier
    ON channels_twilio(webhook_identifier);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_twilio_messaging_service_sid
    ON channels_twilio(messaging_service_sid)
    WHERE messaging_service_sid IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_twilio_account_phone
    ON channels_twilio(account_id, phone_number)
    WHERE phone_number IS NOT NULL;
