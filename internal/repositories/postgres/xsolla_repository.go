package postgres

import (
	"context"
)

func (r *PostgresRepository) SaveTransaction(
	ctx context.Context,
	transactionID int64,
	orderID int,
) (bool, error) {
	tag, err := r.pool.Exec(
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

	return tag.RowsAffected() == 1, nil
}