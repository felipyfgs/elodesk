-- 0022_contact_chatwoot_parity.sql: avatar + blocked on contacts + audit entity index.

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS blocked BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity
    ON audit_logs(account_id, entity_type, entity_id, created_at DESC);
