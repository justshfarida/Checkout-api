package repository

import (
	pagination "checkout-api/internal/controllers/pagination"
	"checkout-api/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemsRepo struct {
	db *pgxpool.Pool
}

func NewItemsRepo(db *pgxpool.Pool) *ItemsRepo {
	return &ItemsRepo{db: db}
}

func (r *ItemsRepo) GetItem(ctx context.Context, id int) (*models.Item, error) {
	var item models.Item
	err := r.db.QueryRow(ctx,
		"select id, name, description, price, stock, created_at from items where id=$1", id,
	).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("item %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("unable to fetch item %d: %w", id, err)
	}
	return &item, nil
}

func (r *ItemsRepo) ListOffset(ctx context.Context, filter models.ItemFilter, page pagination.OffsetPage) ([]*models.Item, int, error) {
	orderCol := "id"
	if col, ok := allowedSortColumns[filter.SortBy]; ok {
		orderCol = col
	}

	var total int
	if err := r.db.QueryRow(ctx, "select count(*) from items").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("unable to count items: %w", err)
	}

	query := fmt.Sprintf(
		"select id, name, description, price, stock, created_at from items order by %s limit $1 offset $2",
		orderCol,
	)

	rows, err := r.db.Query(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to run query on ListOffset: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("unable to scan item row: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating item rows: %w", err)
	}

	return items, total, nil
}

var allowedSortColumns = map[string]string{
	"price": "price",
	"name":  "name",
}

func (r *ItemsRepo) ListCursor(ctx context.Context, filter models.ItemFilter, page pagination.CursorPage) ([]*models.Item, bool, error) {
	where, args := "", []any{}
	if page.After > 0 {
		where = " where id > $1"
		args = append(args, page.After)
	}

	args = append(args, page.Limit+1)
	query := fmt.Sprintf(
		"select id, name, description, price, stock, created_at from items%s order by id limit $%d",
		where, len(args),
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to run query on ListCursor: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("unable to scan item row: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("error iterating item rows: %w", err)
	}

	hasMore := len(items) > page.Limit
	if hasMore {
		items = items[:page.Limit]
	}
	return items, hasMore, nil
}
