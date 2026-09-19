package postgres

import (
	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"
	"context"
	"fmt"
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

	// 1. Try to record the Xsolla transaction.
	tag, err := tx.Exec(
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

	// Already processed this Xsolla transaction.
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	// 2. Decrease stock.
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

	// 3. Mark our order as paid.
	result, err := tx.Exec(
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
		return false, fmt.Errorf(
			"order %d",
			repositories.ErrOrderNotPending, 
			orderID,
		)
	}

	// 4. Everything succeeded.
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}