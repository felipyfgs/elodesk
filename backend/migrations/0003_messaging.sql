-- 0003_messaging.sql: Contacts, Contact Inboxes, Conversations, Messages, Attachments

CREATE TABLE IF NOT EXISTS contacts (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    email TEXT,
    phone_number TEXT,
    identifier TEXT,
    additional_attributes JSONB DEFAULT '{}',
    last_activity_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_contacts_email_account ON contacts(email, account_id) WHERE email IS NOT NULL AND email != '';
CREATE UNIQUE INDEX idx_contacts_phone_account ON contacts(phone_number, account_id) WHERE phone_number IS NOT NULL AND phone_number != '';
CREATE UNIQUE INDEX idx_contacts_identifier_account ON contacts(identifier, account_id) WHERE identifier IS NOT NULL AND identifier != '';
CREATE INDEX idx_contacts_account_id ON contacts(account_id);

CREATE TABLE IF NOT EXISTS contact_inboxes (
    id BIGSERIAL PRIMARY KEY,
    contact_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    inbox_id BIGINT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL,
    hmac_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_contact_inboxes_inbox_source ON contact_inboxes(inbox_id, source_id);
CREATE INDEX idx_contact_inboxes_contact_id ON contact_inboxes(contact_id);
CREATE INDEX idx_contact_inboxes_inbox_id ON contact_inboxes(inbox_id);
CREATE INDEX idx_contact_inboxes_source_id ON contact_inboxes(source_id);

CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    inbox_id BIGINT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    status INTEGER NOT NULL DEFAULT 0,
    assignee_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    contact_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    contact_inbox_id BIGINT REFERENCES contact_inboxes(id) ON DELETE SET NULL,
    display_id BIGSERIAL,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    additional_attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_conversations_account_display ON conversations(account_id, display_id);
CREATE UNIQUE INDEX idx_conversations_uuid ON conversations(uuid);
CREATE INDEX idx_conversations_account_id ON conversations(account_id);
CREATE INDEX idx_conversations_inbox_id ON conversations(inbox_id);
CREATE INDEX idx_conversations_contact_id ON conversations(contact_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_assignee_id ON conversations(assignee_id);

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    inbox_id BIGINT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_type INTEGER NOT NULL DEFAULT 0,
    content_type INTEGER NOT NULL DEFAULT 0,
    content TEXT,
    source_id TEXT,
    private BOOLEAN NOT NULL DEFAULT FALSE,
    status INTEGER DEFAULT 0,
    content_attributes JSONB DEFAULT '{}',
    sender_type TEXT,
    sender_id BIGINT,
    external_source_ids JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_messages_account_id ON messages(account_id);
CREATE INDEX idx_messages_inbox_id ON messages(inbox_id);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_source_id ON messages(source_id) WHERE source_id IS NOT NULL;
CREATE INDEX idx_messages_deleted_at ON messages(deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS attachments (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    file_type INTEGER DEFAULT 0,
    external_url TEXT,
    file_key TEXT,
    extension TEXT,
    meta JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attachments_message_id ON attachments(message_id);
CREATE INDEX idx_attachments_account_id ON attachments(account_id);
