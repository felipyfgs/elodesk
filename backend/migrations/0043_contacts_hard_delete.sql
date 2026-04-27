-- 0043_contacts_hard_delete.sql
-- Reverte a coluna deleted_at de contacts (introduzida em 0041) e relaxa o FK
-- messages.sender_contact_id para ON DELETE SET NULL — sem isso o destroy de
-- um Contact bumpa em mensagens cuja conversation está fora da cadeia de
-- cascade. Conversations seguem em hard-delete (modelo Chatwoot); o experimento
-- de soft-delete em contacts foi abandonado.
--
-- Os índices únicos sobre contacts (phone_number, phone_e164, email,
-- account_lower_email, identifier) já estão na forma correta desde 0003/0011/
-- 0015 — não precisam ser recriados.

DROP INDEX IF EXISTS contacts_deleted_at_idx;
ALTER TABLE contacts DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_sender_contact_id_fkey;
ALTER TABLE messages
    ADD CONSTRAINT messages_sender_contact_id_fkey
    FOREIGN KEY (sender_contact_id) REFERENCES contacts(id) ON DELETE SET NULL;
