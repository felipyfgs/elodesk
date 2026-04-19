-- 0005: KEK encryption + message idempotency

-- Channel::Api api_token is now stored as SHA-256 hash (for lookup) while
-- the plaintext is returned to the user ONCE on inbox creation and never
-- persisted anywhere. The old `api_token` column is no longer read/written
-- by the application but is kept to avoid data loss during rollback.
ALTER TABLE channels_api ADD COLUMN IF NOT EXISTS api_token_hash TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_api_token_hash
    ON channels_api(api_token_hash)
    WHERE api_token_hash IS NOT NULL;

-- hmac_token now stores base64(nonce || AES-256-GCM ciphertext); uniqueness
-- on ciphertext is meaningless (nonces randomize it) so drop the constraint.
ALTER TABLE channels_api DROP CONSTRAINT IF EXISTS channels_api_hmac_token_key;

-- Idempotency: upsert key for inbound messages from providers.
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_inbox_source
    ON messages(inbox_id, source_id)
    WHERE source_id IS NOT NULL;
