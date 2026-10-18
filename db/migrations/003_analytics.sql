-- Exercise 4: fan-out to a second consumer (analytics).
BEGIN;

-- 1. Dedup per consumer. With a fanout exchange the email worker and the
-- analytics worker each get a copy of the SAME message id, so the key must
-- include who processed it. Existing rows belong to the email worker.
ALTER TABLE processed_messages
    ADD COLUMN IF NOT EXISTS consumer TEXT NOT NULL DEFAULT 'email';
ALTER TABLE processed_messages DROP CONSTRAINT IF EXISTS processed_messages_pkey;
ALTER TABLE processed_messages ADD PRIMARY KEY (consumer, event_type, message_id);

-- 2. What the analytics worker builds: paid orders and revenue per day.
CREATE TABLE IF NOT EXISTS daily_sales (
    day        DATE        PRIMARY KEY,
    orders     INTEGER     NOT NULL DEFAULT 0,
    revenue    BIGINT      NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
