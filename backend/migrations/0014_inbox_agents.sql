-- 0014_inbox_agents.sql: junction table linking users to inboxes as agents

CREATE TABLE IF NOT EXISTS inbox_agents (
    id BIGSERIAL PRIMARY KEY,
    inbox_id BIGINT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (inbox_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_inbox_agents_inbox_id ON inbox_agents(inbox_id);
CREATE INDEX IF NOT EXISTS idx_inbox_agents_user_id ON inbox_agents(user_id);
