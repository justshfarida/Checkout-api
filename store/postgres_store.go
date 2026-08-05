package store

import (
	"checkout-api/models"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PostgresStore is an in-memory store for items and orders.
type PostgresStore struct {
	conn *pgx.Conn
}

// NewPostgresStore creates a Store pre-loaded with seed data.
func NewPostgresStore(conn *pgx.Conn) *PostgresStore {
	s := &PostgresStore{
		conn: conn,
	}
	return s
}

// GetItems returns all available items.
func (s *PostgresStore) GetItems(ctx context.Context) ([]*models.Item, error) {
	rows, err := s.conn.Query(ctx, "select * from items")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to run query on GetItems", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
		if err != nil {
			// Handle the scan error, potentially breaking the loop or logging and continuing
			fmt.Printf("unable to scan row: %w", err)
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		items = append(items, &item)
	}
	if rows.Err() != nil {
		return nil, err
	}

	return items, nil
}

// GetItem returns a single item by ID, or nil if not found.
func (s *PostgresStore) GetItem(ctx context.Context, id int) *models.Item {
	// TODO: query a single item with conn.QueryRow()
	row := s.conn.QueryRow(ctx, "select * from items where id=$1", id)
	var item models.Item
	err := row.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		fmt.Printf("unable to scan row: %w", err)
		return nil
	}
	return &item
}

func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) *models.Order {
	// TODO: create an order in a transaction
	// Use a context.Context passed as the first argument from your method
	// Use transaction with conn.Begin(), conn.Exec()
	return nil
}

func (s *PostgresStore) CreateUserCart(ctx context.Context, cart *models.Cart) error {
	// TODO: implement
	tx, err := s.conn.Begin(ctx)

	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO carts (id, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4)", cart.ID, cart.UserID, time.Now(), time.Now())
	if err != nil {
		return err
	}
	query := "INSERT INTO cart_items (cart_id, item_id, quantity, price) VALUES ($1, $2, $3, $4)"
	for _, item := range cart.Items {
		_, err := tx.Exec(ctx, query, cart.ID, item.ItemID, item.Quantity, item.Price)
		if err != nil {
			return fmt.Errorf("failed to insert cart item %d: %w", item.ItemID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
	// _, err=tx.Exec(ctx, "INSERT INTO cart_items (cart_id, item_id, quantity, price) VALUES ($1, $2, $3, $4)", cart.ID. )
}

func (s *PostgresStore) GetUserCart(ctx context.Context, userID int) *models.Cart {
	// TODO: implement

	return nil
}

func (s *PostgresStore) DeleteUserCart(ctx context.Context, userID int) {
	// TODO: implement

}

func (s *PostgresStore) UpdateCartItem(ctx context.Context, userID int, itemID int, quantity int) bool {
	// TODO: implement
	return false
}

func (s *PostgresStore) RemoveCartItem(ctx context.Context, userID int, itemID int) bool {
	// TODO: implement
	// row := s.conn.QueryRow(ctx, "select * from carts where user_id=$1", userID)
	// var cart models.Cart
	// err := row.Scan(&cart.ID, &cart.UserID)
	// if err != nil {
	// 	fmt.Println("unable to scan row: %w", err)
	// }
	// row2 := s.conn.QueryRow(ctx, "delete from cart_items where cart_id=$1 and item_id=$2", cart.ID, itemID)
	query := `
		DELETE FROM cart_items
		USING carts
		WHERE cart_items.cart_id = carts.id
		  AND carts.user_id = $1
		  AND cart_items.item_id = $2
	`

	cmdTag, err := s.conn.Exec(ctx, query, userID, itemID)
	if err != nil {
		fmt.Printf("error removing cart item: %v\n", err)
		return false
	}

	// cmdTag.RowsAffected() returns 1 if item was found and deleted, 0 if it wasn't
	return cmdTag.RowsAffected() > 0
}
