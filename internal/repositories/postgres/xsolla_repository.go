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

	// 3. Decrease stock.
	for _, item := range items {
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
	}

	// 4. Mark order as paid.
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

	// 5. Clear the user's cart.
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

	// 6. Commit everything.
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}