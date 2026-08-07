package store

import (
	"checkout-api/models"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is an in-memory store for items and orders.
// type PostgresStore struct {
// 	conn *pgx.Conn
// }

// // NewPostgresStore creates a Store pre-loaded with seed data.
//
//	func NewPostgresStore(conn *pgx.Conn) *PostgresStore {
//		s := &PostgresStore{
//			conn: conn,
//		}
//		return s
//	}
type PostgresStore struct {
	conn *pgxpool.Pool
}

func NewPostgresStore(conn *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		conn: conn,
	}
}
func (s *PostgresStore) CreateUser(ctx context.Context, username, email, password string) (int, error) {
	var id int
	query := `
		INSERT INTO users (username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id;
	`

	err := s.conn.QueryRow(ctx, query, username, email, password).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetItems returns all available items.
func (s *PostgresStore) GetItems(ctx context.Context) ([]*models.Item, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT id, name, description, price, stock, created_at, updated_at
		FROM items
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to run query on GetItems", err)
	}
	defer rows.Close()

	var items []*models.Item

	for rows.Next() {
		var item models.Item

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// GetItem returns a single item by ID, or nil if not found.
func (s *PostgresStore) GetItem(ctx context.Context, id int) *models.Item {
	row := s.conn.QueryRow(ctx, `
		SELECT id, name, description, price, stock, created_at, updated_at
		FROM items
		WHERE id = $1
	`, id)

	var item models.Item

	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.Stock,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		fmt.Printf("unable to scan row: %v\n", err)
		return nil
	}

	return &item
}
func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert order
	var orderID int
	insertOrderQuery := `
		INSERT INTO orders (user_id, total, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err = tx.QueryRow(ctx, insertOrderQuery, userID, total, status).Scan(&orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	// 2. Insert order items
	insertItemQuery := `
		INSERT INTO order_items (order_id, item_id, price, quantity)
		VALUES ($1, $2, $3, $4)
	`
	for _, item := range items {
		_, err := tx.Exec(ctx, insertItemQuery, orderID, item.ItemID, item.Price, item.Quantity)
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item (item_id %d): %w", item.ItemID, err)
		}
	}

	// 3. Commit order creation (Cart is completely untouched here)
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.Order{
		ID:     orderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}, nil
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

func (s *PostgresStore) GetUserCart(ctx context.Context, userID int) (*models.Cart, error) {
	query := `
		SELECT 
			c.id, 
			c.user_id, 
			ci.item_id, 
			ci.quantity, 
			ci.price
		FROM carts c
		INNER JOIN cart_items ci ON c.id = ci.cart_id
		WHERE c.user_id = $1
	`

	rows, err := s.conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart: %w", err)
	}
	defer rows.Close()

	var cart *models.Cart

	for rows.Next() {
		var (
			cartID     string
			cartUserID int
			item       models.LineItem
		)
		err := rows.Scan(&cartID, &cartUserID, &item.ItemID, &item.Quantity, &item.Price)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if cart == nil {
			cart = &models.Cart{
				ID:     cartID,
				UserID: cartUserID,
				Items:  make([]models.LineItem, 0),
			}
		}

		cart.Items = append(cart.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Returns nil if 0 rows were found (cart does not exist or was deleted)
	return cart, nil
}

func (s *PostgresStore) DeleteUserCart(ctx context.Context, userID int) error {
	query := `DELETE FROM carts WHERE user_id = $1`
	_, err := s.conn.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateCartItem(ctx context.Context, userID int, itemID int, quantity int) bool {
	query := `
		UPDATE cart_items
		SET quantity = $3
		FROM carts
		WHERE cart_items.cart_id = carts.id
		  AND carts.user_id = $1
		  AND cart_items.item_id = $2
	`

	cmdTag, err := s.conn.Exec(ctx, query, userID, itemID, quantity)
	if err != nil {
		fmt.Printf("error updating cart item: %v\n", err)
		return false
	}

	return cmdTag.RowsAffected() > 0
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
func (s *PostgresStore) BuyItemWithoutLock(ctx context.Context, itemID int) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var stock int

	err = tx.QueryRow(ctx,
		`SELECT stock
		 FROM items
		 WHERE id=$1`,
		itemID,
	).Scan(&stock)

	if err != nil {
		return err
	}

	if stock <= 0 {
		return fmt.Errorf("out of stock")
	}

	// simulate slow processing
	//This gives the second goroutine enough time to also read the old value.
	time.Sleep(2 * time.Second)

	_, err = tx.Exec(ctx,
		`UPDATE items
		 SET stock = stock - 1
		 WHERE id=$1`,
		itemID,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func (s *PostgresStore) BuyItemWithLock(ctx context.Context, itemID int) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var stock int
	// Locks that row until the transaction commits or rolls back.
	err = tx.QueryRow(ctx,
		`SELECT stock
		 FROM items
		 WHERE id=$1
		 FOR UPDATE`,
		itemID,
	).Scan(&stock)

	if err != nil {
		return err
	}

	if stock <= 0 {
		return fmt.Errorf("out of stock")
	}

	time.Sleep(2 * time.Second)

	_, err = tx.Exec(ctx,
		`UPDATE items
		 SET stock = stock - 1
		 WHERE id=$1`,
		itemID,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
