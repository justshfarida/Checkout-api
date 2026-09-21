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

	// Already processed.
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

	// 3. Decrease stock only for normal items.
	// Gold packages (credit_amount IS NOT NULL) have unlimited stock.
	for _, item := range items {
		result, err := tx.Exec(
			ctx,
			`
		UPDATE items
		SET stock = stock - $1
		WHERE id = $2
		  AND credit_amount IS NULL
		  AND stock >= $1
		`,
			item.Quantity,
			item.ItemID,
		)
		if err != nil {
			return false, err
		}

		if result.RowsAffected() == 0 {
			// Check whether this is a Gold package.
			var isGoldPackage bool

			err = tx.QueryRow(
				ctx,
				`
			SELECT credit_amount IS NOT NULL
			FROM items
			WHERE id = $1
			`,
				item.ItemID,
			).Scan(&isGoldPackage)
			if err != nil {
				return false, err
			}

			if !isGoldPackage {
				return false, validation.ErrInsufficientStock
			}
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
