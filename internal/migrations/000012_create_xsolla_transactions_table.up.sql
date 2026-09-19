CREATE TABLE xsolla_transactions (
    transaction_id BIGINT PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_xsolla_transactions_order_id
ON xsolla_transactions(order_id);