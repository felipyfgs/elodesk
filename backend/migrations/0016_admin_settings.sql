-- 0016_admin_settings.sql: Agent invitations, macros, SLA policies, audit logs, outbound webhooks

-- Agent invitations
CREATE TABLE IF NOT EXISTS agent_invitations (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role SMALLINT NOT NULL DEFAULT 0,
    name VARCHAR(255),
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_invitations_account_email_pending
    ON agent_invitations (account_id, lower(email))
    WHERE consumed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_invitations_token_hash
    ON agent_invitations (token_hash);

-- Macros
CREATE TABLE IF NOT EXISTS macros (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    visibility VARCHAR(20) NOT NULL DEFAULT 'account',
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_macros_account_id ON macros (account_id);

-- SLA policies
CREATE TABLE IF NOT EXISTS sla_policies (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    first_response_minutes INT NOT NULL DEFAULT 60,
    resolution_minutes INT NOT NULL DEFAULT 1440,
    business_hours_only BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sla_policies_account_id ON sla_policies (account_id);

-- SLA bindings
CREATE TABLE IF NOT EXISTS sla_bindings (
    id BIGSERIAL PRIMARY KEY,
    sla_id BIGINT NOT NULL REFERENCES sla_policies(id) ON DELETE CASCADE,
    inbox_id BIGINT,
    label_id BIGINT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sla_bindings_sla_inbox ON sla_bindings (sla_id, inbox_id) WHERE inbox_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_sla_bindings_sla_label ON sla_bindings (sla_id, label_id) WHERE label_id IS NOT NULL;

-- Add SLA columns to conversations
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS sla_policy_id BIGINT REFERENCES sla_policies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS sla_first_response_due_at TIMESTAMPTZ;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS sla_resolution_due_at TIMESTAMPTZ;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS sla_breached BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_conversations_sla_breached
    ON conversations (account_id, sla_breached) WHERE sla_breached = TRUE;

-- Audit logs (partitioned monthly)
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL,
    account_id BIGINT NOT NULL,
    user_id BIGINT,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100),
    entity_id BIGINT,
    metadata JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create current month partition
CREATE TABLE IF NOT EXISTS audit_logs_202604 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE INDEX IF NOT EXISTS idx_audit_logs_account_created
    ON audit_logs (account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_account_entity
    ON audit_logs (account_id, entity_type, entity_id);

-- Outbound webhooks
CREATE TABLE IF NOT EXISTS outbound_webhooks (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    url VARCHAR(2048) NOT NULL,
    subscriptions JSONB NOT NULL DEFAULT '[]',
    secret VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbound_webhooks_account_id ON outbound_webhooks (account_id);
