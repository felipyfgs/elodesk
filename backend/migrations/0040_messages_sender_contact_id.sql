-- 0040_messages_sender_contact_id.sql: explicit FK to contacts for the
-- author of a group message. Coexists with the polymorphic sender_type/
-- sender_id pair (added in 0003): when the message is in a 1:1 conversation
-- the FK stays NULL and callers fall back to conversation.contact_id; when
-- the message is in a group, this FK identifies the actual member who sent
-- it (sender_type/sender_id then point at the group "proxy" contact).

ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_contact_id BIGINT REFERENCES contacts(id);

CREATE INDEX IF NOT EXISTS messages_sender_contact_id_idx
    ON messages(sender_contact_id) WHERE sender_contact_id IS NOT NULL;
