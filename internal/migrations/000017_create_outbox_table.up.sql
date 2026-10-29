-- Exercise 2: transactional outbox.
-- The payment transaction inserts a row here together with marking the order
-- paid, so the event exists if and only if the payment was committed. The
-- outbox relay publishes unsent rows to RabbitMQ and then sets sent_at.
CREATE TABLE IF NOT EXISTS outbox (
    id          BIGSERIAL   PRIMARY KEY,
    event_type  TEXT        NOT NULL,
    message_id  TEXT        NOT NULL UNIQUE,   -- e.g. order.paid:42, used for dedup
    payload     JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at     TIMESTAMPTZ,                   -- NULL = still waiting to be published
    attempts    INT         NOT NULL DEFAULT 0,
    last_error  TEXT
);

-- The relay only ever looks for unsent rows; keep that lookup fast.
CREATE INDEX IF NOT EXISTS outbox_unsent_idx ON outbox (id) WHERE sent_at IS NULL;
