BEGIN;

ALTER TABLE items DROP CONSTRAINT IF EXISTS items_item_type_check;

ALTER TABLE items
    DROP COLUMN IF EXISTS item_type,
    DROP COLUMN IF EXISTS credit_amount;

COMMIT;
