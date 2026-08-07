BEGIN;

CREATE OR REPLACE PROCEDURE increase_item_stock(
    p_item_id INTEGER,
    p_quantity INTEGER
)
LANGUAGE plpgsql
AS $$
BEGIN
    IF p_quantity <= 0 THEN
        RAISE EXCEPTION 'Quantity must be greater than 0';
    END IF;

    UPDATE items
    SET stock = stock + p_quantity
    WHERE id = p_item_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Item with id % does not exist', p_item_id;
    END IF;
END;
$$;

COMMIT;