-- 0015_contacts_email_unique_idx.sql: Case-insensitive unique email per account
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_account_lower_email
    ON contacts (account_id, lower(email))
    WHERE email IS NOT NULL AND email != '';
