BEGIN;

ALTER TABLE items
    ADD COLUMN IF NOT EXISTS credit_amount INTEGER,
    ADD COLUMN IF NOT EXISTS item_type     VARCHAR(30) NOT NULL DEFAULT 'merch';

ALTER TABLE items DROP CONSTRAINT IF EXISTS items_item_type_check;

ALTER TABLE items
    ADD CONSTRAINT items_item_type_check
    CHECK (item_type IN ('merch', 'currency_package', 'virtual_item'));

COMMIT;
