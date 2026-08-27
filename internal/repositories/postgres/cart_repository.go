package postgres

import (
	"context"
	"fmt"

	"checkout-api/internal/models"
)

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

func (r *PostgresRepository) GetUserCart(
	ctx context.Context,
	userID int,
) ([]models.CartItem, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			i.id,
			i.name,
			i.description,
			i.price,
			i.stock,
			i.created_at,
			c.quantity
		FROM carts c
		INNER JOIN items i
			ON i.id = c.item_id
		WHERE c.user_id = $1
		ORDER BY i.id
		`,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get cart for user %d: %w",
			userID,
			err,
		)
	}

	defer rows.Close()

	items := make([]models.CartItem, 0)

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
		); err != nil {
			return nil, fmt.Errorf(
				"failed to scan cart item: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed while reading cart: %w",
			err,
		)
	}

	return items, nil
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
