-- 0041_contacts_deleted_at.sql: soft-delete support for contacts.
-- Instead of hard DELETE, contacts are flagged with deleted_at.
-- Existing queries should add WHERE deleted_at IS NULL to exclude soft-deleted rows.

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS contacts_deleted_at_idx ON contacts(deleted_at) WHERE deleted_at IS NULL;
