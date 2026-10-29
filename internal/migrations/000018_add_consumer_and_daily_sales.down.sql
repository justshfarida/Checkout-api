BEGIN;

DROP TABLE IF EXISTS daily_sales;

-- Back to one dedup row per message: keep only the email worker's rows.
ALTER TABLE processed_messages DROP CONSTRAINT IF EXISTS processed_messages_pkey;
DELETE FROM processed_messages WHERE consumer <> 'email';
ALTER TABLE processed_messages DROP COLUMN IF EXISTS consumer;
ALTER TABLE processed_messages ADD PRIMARY KEY (event_type, message_id);

COMMIT;
