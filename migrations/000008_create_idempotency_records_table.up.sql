BEGIN;

CREATE TABLE idempotency_records (
    key text primary key,
    status_code INTEGER NOT NULL,
    body  TEXT NOT NULL,
    request_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMIT;
