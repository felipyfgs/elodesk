-- 0006_helpdesk_core.sql: Helpdesk operational layer tables
-- Labels, label_taggings, teams, team_members, canned_responses,
-- notes, custom_attribute_definitions, custom_filters
-- plus team_id on conversations and GIN indexes

-- Labels
CREATE TABLE IF NOT EXISTS labels (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#1f93ff',
    description TEXT,
    show_on_sidebar BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (account_id, lower(title))
);

CREATE INDEX IF NOT EXISTS idx_labels_account ON labels(account_id);

-- Label Taggings (polymorphic)
CREATE TABLE IF NOT EXISTS label_taggings (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    taggable_type TEXT NOT NULL CHECK (taggable_type IN ('conversation', 'contact')),
    taggable_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (label_id, taggable_type, taggable_id)
);

CREATE INDEX IF NOT EXISTS idx_label_taggings_taggable ON label_taggings(taggable_type, taggable_id);
CREATE INDEX IF NOT EXISTS idx_label_taggings_account ON label_taggings(account_id);

-- Teams
CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    allow_auto_assign BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (account_id, lower(name))
);

CREATE INDEX IF NOT EXISTS idx_teams_account ON teams(account_id);

-- Team Members
CREATE TABLE IF NOT EXISTS team_members (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (team_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);

-- Canned Responses
CREATE TABLE IF NOT EXISTS canned_responses (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    short_code TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (account_id, short_code)
);

CREATE INDEX IF NOT EXISTS idx_canned_responses_account ON canned_responses(account_id);

-- Notes
CREATE TABLE IF NOT EXISTS notes (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    contact_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_contact ON notes(contact_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_account ON notes(account_id);

-- Custom Attribute Definitions
CREATE TABLE IF NOT EXISTS custom_attribute_definitions (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    attribute_key TEXT NOT NULL,
    attribute_display_name TEXT NOT NULL,
    attribute_display_type TEXT NOT NULL CHECK (attribute_display_type IN ('text', 'number', 'currency', 'percent', 'link', 'date', 'list', 'checkbox')),
    attribute_model TEXT NOT NULL CHECK (attribute_model IN ('contact', 'conversation')),
    attribute_values JSONB,
    attribute_description TEXT,
    regex_pattern TEXT,
    default_value TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (account_id, attribute_model, attribute_key)
);

CREATE INDEX IF NOT EXISTS idx_custom_attr_defs_account ON custom_attribute_definitions(account_id);

-- Custom Filters (Saved Filters)
CREATE TABLE IF NOT EXISTS custom_filters (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    filter_type TEXT NOT NULL CHECK (filter_type IN ('conversation', 'contact')),
    query JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_filters_user_account ON custom_filters(user_id, account_id);

-- team_id on conversations
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS team_id BIGINT NULL REFERENCES teams(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_conversations_team ON conversations(team_id) WHERE team_id IS NOT NULL;

-- GIN indexes for JSONB additional_attributes (used by custom attributes + saved filters)
CREATE INDEX IF NOT EXISTS idx_contacts_additional_attrs_gin ON contacts USING GIN (additional_attributes);
CREATE INDEX IF NOT EXISTS idx_conversations_additional_attrs_gin ON conversations USING GIN (additional_attributes);
