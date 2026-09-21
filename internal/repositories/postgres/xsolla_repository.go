package postgres

import (
	"context"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"
)

func (r *PostgresRepository) ProcessXsollaPayment(
	ctx context.Context,
	transactionID int64,
	orderID int,
	items []models.LineItem,
) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// 1. Idempotency
	result, err := tx.Exec(
		ctx,
		`
		INSERT INTO xsolla_transactions (
			transaction_id,
			order_id
		)
		VALUES ($1, $2)
		ON CONFLICT (transaction_id) DO NOTHING
		`,
		transactionID,
		orderID,
	)
	if err != nil {
		return false, err
	}

	// This transaction was already processed.
	if result.RowsAffected() == 0 {
		return false, nil
	}

	// 2. Get and lock the order.
	var (
		userID int
		status string
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			user_id,
			status
		FROM orders
		WHERE id = $1
		FOR UPDATE
		`,
		orderID,
	).Scan(
		&userID,
		&status,
	)
	if err != nil {
		return false, err
	}

	if status != "pending" {
		return false, repositories.ErrOrderNotPending
	}

	// 3. Process each item according to its type.
	for _, item := range items {
		var itemType string

		err = tx.QueryRow(
			ctx,
			`
			SELECT item_type
			FROM items
			WHERE id = $1
			`,
			item.ItemID,
		).Scan(&itemType)
		if err != nil {
			return false, err
		}

		switch itemType {

		case "merch":
			// Physical merchandise has limited stock.
			result, err := tx.Exec(
				ctx,
				`
				UPDATE items
				SET stock = stock - $1
				WHERE id = $2
				  AND stock >= $1
				`,
				item.Quantity,
				item.ItemID,
			)
			if err != nil {
				return false, err
			}

			if result.RowsAffected() == 0 {
				return false, validation.ErrInsufficientStock
			}

		case "virtual_item":
			// Give the purchased virtual item to the user.
			_, err := tx.Exec(
				ctx,
				`
				INSERT INTO user_inventory (
					user_id,
					item_id,
					quantity
				)
				VALUES ($1, $2, $3)
				ON CONFLICT (user_id, item_id)
				DO UPDATE SET
					quantity = user_inventory.quantity + EXCLUDED.quantity
				`,
				userID,
				item.ItemID,
				item.Quantity,
			)
			if err != nil {
				return false, err
			}

		case "currency_package":
			// No stock handling here.
			// Gold is credited below using credit_amount.
		}
	}

	// 4. Calculate how much Gold this order gives.
	var totalCredit int

	err = tx.QueryRow(
		ctx,
		`
		SELECT COALESCE(
			SUM(COALESCE(i.credit_amount, 0) * li.quantity),
			0
		)
		FROM line_items li
		JOIN items i ON i.id = li.item_id
		WHERE li.order_id = $1
		`,
		orderID,
	).Scan(&totalCredit)
	if err != nil {
		return false, err
	}

	// 5. Credit Gold to the user's balance.
	if totalCredit > 0 {
		_, err = tx.Exec(
			ctx,
			`
			UPDATE users
			SET balance = balance + $1
			WHERE id = $2
			`,
			totalCredit,
			userID,
		)
		if err != nil {
			return false, err
		}
	}

	// 6. Mark order as paid.
	result, err = tx.Exec(
		ctx,
		`
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
		  AND status = 'pending'
		`,
		orderID,
	)
	if err != nil {
		return false, err
	}

	if result.RowsAffected() == 0 {
		return false, repositories.ErrOrderNotPending
	}

	// 7. Clear the user's cart.
	_, err = tx.Exec(
		ctx,
		`
		DELETE FROM carts
		WHERE user_id = $1
		`,
		userID,
	)
	if err != nil {
		return false, err
	}

	// 8. Commit everything.
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}
