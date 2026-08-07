package store

import (
	"checkout-api/models"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostgresStore struct {
	conn *pgx.Conn
}

func NewPostgresStore(conn *pgx.Conn) *PostgresStore {
	return &PostgresStore{conn: conn}
}

func (s *PostgresStore) DB() *Query {
	return &Query{
		DBTX: s.conn,
	}
}

func (s *PostgresStore) WithTx(tx pgx.Tx) *Query {
	return &Query{
		DBTX: tx,
	}
}

// Ietms

func (s *PostgresStore) GetItems(ctx context.Context) ([]*models.Item, error) {
	rows, err := s.DB().GetItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	items := make([]*models.Item, 0)
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.Price, &it.Stock, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, &it)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetItem(ctx context.Context, id int) (*models.Item, error) {
	var item models.Item
	err := s.DB().GetItem(ctx, id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item %d: %w", id, err)
	}
	return &item, nil
}

// Users

func (s *PostgresStore) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	row := s.DB().GetUserByEmail(ctx, email)
	err := row.Scan(&user.ID, &user.Email, &user.Hash)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *PostgresStore) SaveUser(ctx context.Context, email string, hash []byte) error {
	_, err := s.DB().InsertUser(ctx, email, hash)
	return err
}

// Cart

func (s *PostgresStore) UpsertCartItem(ctx context.Context, userID int, itemID int, quantity int) error {
	_, err := s.DB().UpsertCart(ctx, userID, itemID, quantity)
	return err
}

func (s *PostgresStore) GetUserCart(ctx context.Context, userID int) ([]models.CartItem, error) {
	rows, err := s.DB().GetItemsFromUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart for user %d: %w", userID, err)
	}
	defer rows.Close()

	items := make([]models.CartItem, 0)
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt, &item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan cart row: %w", err)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil
}

func (s *PostgresStore) RemoveCartItem(ctx context.Context, userID int, itemID int) error {
	_, err := s.DB().DeleteItemFromUserCart(ctx, userID, itemID)
	return err
}

func (s *PostgresStore) DeleteUserCart(ctx context.Context, userID int) error {
	_, err := s.DB().DeleteCartByUserID(ctx, userID)
	return err
}

// Orders

func (s *PostgresStore) CreateOrder(ctx context.Context, idempotencyKey string, userID int, items []models.LineItem, total int, status string) (*models.Order, bool, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := s.WithTx(tx)

	// Pessimistic lock: acquire row-level lock on each item and verify sufficient stock.
	for _, lineItem := range items {
		var item models.Item
		err := q.GetItemByIDForUpdate(ctx, lineItem.ItemID).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)		if err != nil {
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, false, fmt.Errorf("item %d not found", lineItem.ItemID)
				}
				return nil, false, fmt.Errorf("failed to lock item %d: %w", lineItem.ItemID, err)
			}
			if item.Stock < lineItem.Quantity {
				return nil, false, fmt.Errorf("Insufficient stock", lineItem.ItemID, err)
			}
	}

	var orderID int
	var createdAt time.Time
	// err = q.InsertOrderReturning(ctx, userID, total, status, idempotencyKey).Scan(&orderID, &createdAt)

	/*
		if errors.Is(err, pgx.ErrNoRows) {
			existing, ferr := s.getOrderByIdempotencyKey(ctx, idempotencyKey)
			if ferr != nil {
				return nil, false, ferr
			}
			return existing, true, nil
		}
		if err != nil {
			return nil, false, err
		}
	*/
	for _, lineItem := range items {
		if _, err := q.InsertLineItem(ctx, orderID, lineItem.ItemID, lineItem.Price, lineItem.Quantity); err != nil {
			return nil, false, fmt.Errorf("failed to insert line item %d: %w", lineItem.ItemID, err)
		}
		if _, err := q.DecrementItemStock(ctx, lineItem.ItemID, lineItem.Quantity); err != nil {
			return nil, false, fmt.Errorf("failed to decrement item stock %d: %w", lineItem.ItemID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}

	return &models.Order{ID: orderID, UserID: userID, Items: items, TotalPrice: total, Status: status, CreatedAt: createdAt}, false, nil
}
/*
func (s *PostgresStore) getOrderItems(ctx context.Context, orderID int) ([]models.LineItem, error) {
	rows, err := s.conn.Query(ctx, `SELECT item_id, quantity, price FROM order_items WHERE order_id=$1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.LineItem
	for rows.Next() {
		var li models.LineItem
		if err := rows.Scan(&li.ItemID, &li.Quantity, &li.Price); err != nil {
			return nil, err
		}
		items = append(items, li)
	}
	return items, rows.Err()
}


func (s *PostgresStore) GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT id, user_id, total_price, status, created_at FROM orders WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for user %d: %w", userID, err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalPrice, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, o := range orders {
		items, err := s.getOrderItems(ctx, o.ID)
		if err != nil {
			return nil, err
		}
		o.Items = items
	}
	return orders, nil
}
*/

// Refresh Tokens

func (s *PostgresStore) SaveRefreshToken(ctx context.Context, userID int, token string) error {
	_, err := s.DB().InsertRefreshToken(ctx, userID, token)
	return err
}

func (s *PostgresStore) FindRefreshToken(ctx context.Context, token string) (int, bool, error) {
	var userID int
	var active bool

	err := s.DB().GetRefreshToken(ctx, token).Scan(&userID, &active)
	if err != nil {
		return 0, false, err
	}

	return userID, active, nil
}

func (s *PostgresStore) DeactivateRefreshToken(ctx context.Context, token string) error {
	_, err := s.DB().DeactivateRefreshToken(ctx, token)
	return err
}
