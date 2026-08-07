BEGIN;

CREATE OR REPLACE FUNCTION is_item_in_stock(p_item_id INTEGER)
RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    item_stock INTEGER;
BEGIN
    SELECT stock
    INTO item_stock
    FROM items
    WHERE id = p_item_id;

    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    RETURN item_stock > 0;
END;
$$;

COMMIT;