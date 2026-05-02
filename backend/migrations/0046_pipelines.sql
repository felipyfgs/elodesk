-- 0046_pipelines.sql: Kanban pipelines with templates, stages, cards
-- Stores account-scoped pipelines (Sales CRM, Support, Tasks, etc),
-- their stages (columns), cards inside stages with optional links to
-- existing Contacts or Conversations, plus M..M assignees and labels.

-- Pipelines (one per account, multiple per account)
CREATE TABLE IF NOT EXISTS pipelines (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    template_key TEXT,
    card_kind SMALLINT NOT NULL DEFAULT 0,
    icon TEXT,
    color TEXT NOT NULL DEFAULT '#1f93ff',
    archived_at TIMESTAMPTZ,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipelines_account_active ON pipelines(account_id) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pipelines_account ON pipelines(account_id);

-- Stages (columns of a pipeline)
CREATE TABLE IF NOT EXISTS pipeline_stages (
    id BIGSERIAL PRIMARY KEY,
    pipeline_id BIGINT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position DOUBLE PRECISION NOT NULL,
    color TEXT NOT NULL DEFAULT '#94a3b8',
    is_terminal BOOLEAN NOT NULL DEFAULT false,
    terminal_kind SMALLINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline_position ON pipeline_stages(pipeline_id, position);

-- Cards (one per row inside a stage)
CREATE TABLE IF NOT EXISTS pipeline_cards (
    id BIGSERIAL PRIMARY KEY,
    pipeline_id BIGINT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES pipeline_stages(id) ON DELETE RESTRICT,
    position DOUBLE PRECISION NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    value_cents BIGINT,
    value_currency CHAR(3),
    due_date DATE,
    custom_attrs JSONB NOT NULL DEFAULT '{}'::jsonb,
    linked_entity_type SMALLINT,
    linked_entity_id BIGINT,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_cards_stage_position ON pipeline_cards(stage_id, position);
CREATE INDEX IF NOT EXISTS idx_pipeline_cards_pipeline ON pipeline_cards(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_cards_link ON pipeline_cards(linked_entity_type, linked_entity_id) WHERE linked_entity_type IS NOT NULL;

-- Assignees (M..M card <-> user)
CREATE TABLE IF NOT EXISTS pipeline_card_assignees (
    card_id BIGINT NOT NULL REFERENCES pipeline_cards(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (card_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_card_assignees_user ON pipeline_card_assignees(user_id);

-- Labels (M..M card <-> existing Label, reuses labels table)
CREATE TABLE IF NOT EXISTS pipeline_card_labels (
    card_id BIGINT NOT NULL REFERENCES pipeline_cards(id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_card_labels_label ON pipeline_card_labels(label_id);
