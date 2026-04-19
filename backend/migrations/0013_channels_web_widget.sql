-- 0012_channels_web_widget.sql: Web Widget channel table + pubsub_token on conversations

CREATE TABLE IF NOT EXISTS channels_web_widget (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    inbox_id BIGINT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    website_token TEXT NOT NULL,
    hmac_token_ciphertext TEXT NOT NULL,
    website_url TEXT NOT NULL DEFAULT '',
    widget_color TEXT NOT NULL DEFAULT '#0084FF',
    welcome_title TEXT NOT NULL DEFAULT '',
    welcome_tagline TEXT NOT NULL DEFAULT '',
    reply_time TEXT NOT NULL DEFAULT 'in_a_few_minutes',
    feature_flags JSONB NOT NULL DEFAULT '{"attachments":true,"emoji_picker":true,"end_conversation":false}',
    requires_reauth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT channels_web_widget_reply_time_check CHECK (reply_time IN ('in_a_few_minutes','in_a_few_hours','in_a_day'))
);

CREATE INDEX IF NOT EXISTS idx_channels_web_widget_account_id ON channels_web_widget(account_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_web_widget_website_token ON channels_web_widget(website_token);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'conversations' AND column_name = 'pubsub_token'
    ) THEN
        ALTER TABLE conversations ADD COLUMN pubsub_token TEXT;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_pubsub_token ON conversations(pubsub_token) WHERE pubsub_token IS NOT NULL;
