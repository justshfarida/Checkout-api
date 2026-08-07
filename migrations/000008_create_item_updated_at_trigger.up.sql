BEGIN;

ALTER TABLE items
ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT NOW();

CREATE OR REPLACE FUNCTION update_item_timestamp()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_update_item_timestamp
BEFORE UPDATE ON items
FOR EACH ROW
EXECUTE FUNCTION update_item_timestamp();

COMMIT;