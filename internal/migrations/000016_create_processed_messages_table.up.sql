-- Exercise 1: idempotent consumer.
-- One row per message a worker has started. processed_at is set only after
-- the side effect (the email) succeeded, so a failed attempt is retried while
-- a finished one is recognised as a duplicate.
CREATE TABLE IF NOT EXISTS processed_messages (
    event_type   TEXT        NOT NULL,
    message_id   TEXT        NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    PRIMARY KEY (event_type, message_id)
);
