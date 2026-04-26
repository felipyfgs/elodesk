-- 0038_contacts_avatar_hash.sql: SHA-256 of the avatar payload, used by
-- ContactService.UpsertContact to skip re-downloads when the upstream URL
-- changes but the binary content is identical. last_activity_at and
-- avatar_url already exist (see 0003 and 0022).

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS avatar_hash TEXT;
