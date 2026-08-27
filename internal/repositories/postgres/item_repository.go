package postgres

import (
	"context"
	"errors"
	"fmt"

	"checkout-api/internal/models"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetItems(
	ctx context.Context,
) ([]*models.Item, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			name,
			description,
			price,
			stock,
			created_at
		FROM items
		ORDER BY id
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get items: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.Item, 0)

	for rows.Next() {
		var item models.Item

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to scan item: %w",
				err,
			)
		}

		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed while reading items: %w",
			err,
		)
	}

	return items, nil
}

func (r *PostgresRepository) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	var item models.Item

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			name,
			description,
			price,
			stock,
			created_at
		FROM items
		WHERE id = $1
		`,
		id,
	).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.Stock,
		&item.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"failed to get item %d: %w",
			id,
			err,
		)
	}

	return &item, nil
}
