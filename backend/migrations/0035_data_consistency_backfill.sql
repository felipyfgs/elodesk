-- 0035_data_consistency_backfill.sql: one-shot backfill for derived fields
-- that were never populated due to bugs in the message/contact/conversation
-- write paths (fixed in code in this same change). Idempotent — guarded by
-- IS NULL / IS DISTINCT FROM, so re-running on already-clean data is a no-op.

-- 1. messages.sender — incoming attributed to the conversation contact;
-- outgoing/template/activity attributed to the highest-privileged
-- account_user (Owner=2 ranks above Admin=1 and Agent=0; ties broken by id
-- to keep the choice deterministic across reruns).
UPDATE messages m
   SET sender_type = 'Contact',
       sender_id = c.contact_id
  FROM conversations c
 WHERE m.conversation_id = c.id
   AND m.sender_id IS NULL
   AND m.message_type = 0;

UPDATE messages m
   SET sender_type = 'User',
       sender_id = (
           SELECT au.user_id
             FROM account_users au
            WHERE au.account_id = m.account_id
            ORDER BY au.role DESC, au.id ASC
            LIMIT 1
       )
 WHERE m.sender_id IS NULL
   AND m.message_type IN (1, 2, 3);

-- 2. conversations.last_activity_at = MAX(messages.created_at) per conv.
UPDATE conversations c
   SET last_activity_at = mx.max_at
  FROM (
      SELECT conversation_id, MAX(created_at) AS max_at
        FROM messages
       GROUP BY conversation_id
  ) mx
 WHERE c.id = mx.conversation_id
   AND c.last_activity_at < mx.max_at;

-- 3. contacts.last_activity_at = MAX(incoming msg.created_at) per contact.
UPDATE contacts ct
   SET last_activity_at = mx.max_at
  FROM (
      SELECT c.contact_id, MAX(msg.created_at) AS max_at
        FROM messages msg
        JOIN conversations c ON c.id = msg.conversation_id
       WHERE msg.message_type = 0
       GROUP BY c.contact_id
  ) mx
 WHERE ct.id = mx.contact_id
   AND ct.last_activity_at IS DISTINCT FROM mx.max_at;

-- 4. contacts.phone_e164 — backfill rows whose phone_number is already in
-- E.164 form (leading + then 1..15 digits). Anything else is left alone:
-- proper normalization needs libphonenumber, which only the Go layer has.
UPDATE contacts
   SET phone_e164 = phone_number
 WHERE phone_e164 IS NULL
   AND phone_number ~ '^\+[1-9][0-9]{1,14}$';
