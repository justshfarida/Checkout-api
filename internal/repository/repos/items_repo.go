package repos

// ONLY IMPORTS ENTITES/CORE/MODELS
import (
	"checkout-api/internal/models"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemsRepo struct {
	db *pgxpool.Pool
}

func NewItemsRepo(db *pgxpool.Pool) *ItemsRepo {
	r := &ItemsRepo{
		db: db,
	}
	return r
}

func (r *ItemsRepo) GetItems(ctx context.Context) ([]*models.Item, error) {
	rows, err := r.db.Query(ctx, "select id, name, description, price, stock, created_at from items")
	if err != nil {
		return nil, fmt.Errorf("failed to run query on GetItems: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
		if err != nil {
			fmt.Printf("unable to scan row: %w", err)
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		items = append(items, &item)
	}
	if rows.Err() != nil {
		return nil, err
	}
	// RETURN PROPER ERROR MESSAGE
	return items, nil
}

func (r *ItemsRepo) GetItem(ctx context.Context, id int) (*models.Item, error) {
	row := r.db.QueryRow(ctx, "select id, name, description, price, stock, created_at from items where id=$1", id)

	var item models.Item
	err := row.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("item %d not found", id)
		}
		return nil, fmt.Errorf("unable to scan row for item %d: %w", id, err)
	}

	return &item, nil
}
