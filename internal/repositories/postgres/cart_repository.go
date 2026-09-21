package postgres

import (
	"context"
	"fmt"

	"checkout-api/internal/models"
)

func (r *PostgresRepository) GetUserCartOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.CartItem, error) {
	rows, err :=
		r.pool.Query(
			ctx,
			`
			SELECT
				i.id,
				i.name,
				i.description,
				i.price,
				i.stock,
				i.created_at,
				c.quantity,
				i.credit_amount, 
				i.item_type
			FROM carts c
			INNER JOIN items i
				ON i.id = c.item_id
			WHERE c.user_id = $1
			ORDER BY i.id
			LIMIT $2
			OFFSET $3
			`,
			userID,
			limit,
			offset,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get user cart: %w",
			err,
		)
	}

	defer rows.Close()

	items :=
		make([]models.CartItem, 0)

	for rows.Next() {
		var item models.CartItem

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
			&item.Quantity,
			&item.CreditAmount,
			&item.ItemType,
		); err != nil {
			return nil, err
		}

		items = append(
			items,
			item,
		)
	}

	return items, rows.Err()
}
func (r *PostgresRepository) GetUserCartCursor(
	ctx context.Context,
	userID int,
	lastID int,
	limit int,
) ([]models.CartItem, error) {
	rows, err :=
		r.pool.Query(
			ctx,
			`
			SELECT
				i.id,
				i.name,
				i.description,
				i.price,
				i.stock,
				i.created_at,
				c.quantity,
				i.credit_amount,
				i.item_type
			FROM carts c
			INNER JOIN items i
				ON i.id = c.item_id
			WHERE c.user_id = $1
			  AND i.id > $2
			ORDER BY i.id
			LIMIT $3
			`,
			userID,
			lastID,
			limit,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get user cart: %w",
			err,
		)
	}

	defer rows.Close()

	items :=
		make([]models.CartItem, 0)

	for rows.Next() {
		var item models.CartItem

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
			&item.Quantity,
			&item.CreditAmount,
			&item.ItemType,
		); err != nil {
			return nil, err
		}

		items = append(
			items,
			item,
		)
	}

	return items, rows.Err()
}
func (r *PostgresRepository) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO carts (
			user_id,
			item_id,
			quantity
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, item_id)
		DO UPDATE
		SET quantity = EXCLUDED.quantity
		`,
		userID,
		itemID,
		quantity,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to upsert cart item: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresRepository) DeleteUserCart(
	ctx context.Context,
	userID int,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		DELETE FROM carts
		WHERE user_id = $1
		`,
		userID,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to delete cart: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresRepository) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		DELETE FROM carts
		WHERE user_id = $1
		  AND item_id = $2
		`,
		userID,
		itemID,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to remove cart item: %w",
			err,
		)
	}

	return nil
}
func (r *PostgresRepository) GetUserCartTotal(
	ctx context.Context,
	userID int,
) (int, error) {
	var total int

	//coalesce returns first non null character. useful for empty cart. sum returns null when cart is empty.
	err := r.pool.QueryRow(
		ctx,

		`
		SELECT COALESCE( 
			SUM(i.price * c.quantity),
			0
		)
		FROM carts c
		INNER JOIN items i
			ON i.id = c.item_id
		WHERE c.user_id = $1
		`,
		userID,
	).Scan(&total)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to calculate cart total: %w",
			err,
		)
	}

	return total, nil
}
