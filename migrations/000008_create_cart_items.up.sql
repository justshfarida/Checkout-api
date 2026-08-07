BEGIN;

CREATE TABLE cart_items (
                            id       SERIAL PRIMARY KEY,
                            cart_id  INTEGER NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
                            item_id  INTEGER NOT NULL REFERENCES items(id),
                            quantity INTEGER NOT NULL CHECK (quantity > 0),
                            UNIQUE (cart_id, item_id)
);

CREATE INDEX idx_cart_items_cart_id ON cart_items (cart_id);

COMMIT;