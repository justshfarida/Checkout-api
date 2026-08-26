package repository

import (
	"checkout-api/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	r := &Repository{
		db: db,
	}
	return r
}

// Items

func (r *Repository) GetItems(ctx context.Context) ([]*models.Item, error) {
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

func (r *Repository) GetItem(ctx context.Context, id int) (*models.Item, error) {
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

// Cart

func (r *Repository) CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var cartID int
	err = tx.QueryRow(ctx,
		`insert into carts (user_id, last_accessed) values ($1, now())
		 on conflict (user_id) do update set last_accessed=now()
		 returning id`,
		cart.UserID,
	).Scan(&cartID)
	if err != nil {
		return fmt.Errorf("unable to create/find cart for user %d: %w", cart.UserID, err)
	}

	_, err = tx.Exec(ctx, `delete from cart_items where cart_id=$1`, cartID)
	if err != nil {
		return fmt.Errorf("unable to clear existing cart items: %w", err)
	}
	tag, err := tx.Exec(ctx, `delete from cart_items where cart_id=$1`, cartID)
	if err != nil {
		return fmt.Errorf("unable to clear existing cart items: %w", err)
	}
	fmt.Printf("cleared %d old cart items for cart %d\n", tag.RowsAffected(), cartID)

	for _, item := range cart.Items {
		_, err := tx.Exec(ctx,
			`insert into cart_items (cart_id, item_id, quantity) values ($1, $2, $3)`,
			cartID, item.ItemID, item.Quantity,
		)
		if err != nil {
			return fmt.Errorf("unable to insert cart item %d: %w", item.ItemID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("unable to commit transaction: %w", err)
	}
	return nil
}

func (r *Repository) GetUserCart(ctx context.Context, userID int) (*models.Cart, error) {
	var cart models.Cart
	err := r.db.QueryRow(ctx,
		"select id, user_id, last_accessed from carts where user_id=$1", userID,
	).Scan(&cart.ID, &cart.UserID, &cart.LastAccessed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to fetch cart for user %d: %w", userID, err)
	}

	rows, err := r.db.Query(ctx,
		`select ci.item_id, ci.quantity, i.name, i.price
         from cart_items ci
         join items i on i.id = ci.item_id
         where ci.cart_id=$1`, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to run query on GetUserCart items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var li models.LineItem
		if err := rows.Scan(&li.ItemID, &li.Quantity, &li.Name, &li.Price); err != nil {
			return nil, fmt.Errorf("unable to scan cart item row: %w", err)
		}
		cart.Items = append(cart.Items, li)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cart item rows: %w", err)
	}

	return &cart, nil
}

func (r *Repository) DeleteUserCart(ctx context.Context, userID int) error {
	_, err := r.db.Exec(ctx, "delete from carts where user_id=$1", userID)
	if err != nil {
		fmt.Printf("unable to delete user cart %d: %w", userID, err)
	}

	return nil
}

func (r *Repository) UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`update cart_items set quantity=$1
		 where item_id=$2 and cart_id = (select id from carts where user_id=$3)`,
		quantity, itemID, userID)
	if err != nil {
		return false, fmt.Errorf("unable to update cart item %d for user %d: %w", itemID, userID, err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`delete from cart_items
		 where item_id=$1 and cart_id = (select id from carts where user_id=$2)`,
		itemID, userID)
	if err != nil {
		return false, fmt.Errorf("unable to remove cart item %d for user %d: %w", itemID, userID, err)
	}
	return tag.RowsAffected() > 0, nil
}

// Order

func (r *Repository) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	order := &models.Order{UserID: userID, Items: items, TotalPrice: total, Status: status}

	err = tx.QueryRow(ctx,
		`insert into orders (user_id, total, status) values ($1, $2, $3) returning id`,
		userID, total, status,
	).Scan(&order.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to create order for user %d: %w", userID, err)
	}

	for _, item := range items {
		_, err := tx.Exec(ctx,
			`insert into order_items (order_id, item_id, price, quantity) values ($1, $2, $3, $4)`,
			order.ID, item.ItemID, item.Price, item.Quantity,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to insert order item %d: %w", item.ItemID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("unable to commit transaction: %w", err)
	}
	return order, nil
}

func (r *Repository) GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error) {
	rows, err := r.db.Query(ctx,
		"select id, user_id, total, status from orders where user_id=$1", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to run query on GetOrdersByUserID: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.TotalPrice, &order.Status); err != nil {
			return nil, fmt.Errorf("unable to scan order row: %w", err)
		}
		orders = append(orders, &order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	for _, order := range orders {
		itemRows, err := r.db.Query(ctx,
			`select oi.item_id, oi.price, oi.quantity, i.name
             from order_items oi
             join items i on i.id = oi.item_id
             where oi.order_id=$1`, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch items for order %d: %w", order.ID, err)
		}

		for itemRows.Next() {
			var li models.LineItem
			if err := itemRows.Scan(&li.ItemID, &li.Price, &li.Quantity, &li.Name); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("unable to scan order item row: %w", err)
			}
			order.Items = append(order.Items, li)
		}
		if err := itemRows.Err(); err != nil {
			itemRows.Close()
			return nil, fmt.Errorf("error iterating order item rows: %w", err)
		}
		itemRows.Close()
	}

	return orders, nil
}

// Users

func (r *Repository) SignUp(ctx context.Context, username string, passwordHash string) (*models.User, error) {
	var user models.User
	user.Username = username
	user.PasswordHash = passwordHash

	err := r.db.QueryRow(ctx, `insert into users (username, password_hash) values ($1, $2) returning id, created_at`, username, passwordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("unable to create/find cart for user %d: %w")
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // username alr exists err
			return nil, fmt.Errorf("username %q already taken", username)
		}
		return nil, fmt.Errorf("unable to create user: %w", err)
	}
	return &user, nil
}

func (r *Repository) Login(ctx context.Context, username, password string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx,
		"select id, username, password_hash, created_at from users where username=$1", username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("invalid username or password")
		}
		return nil, fmt.Errorf("unable to fetch user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	return &user, nil
}

func (r *Repository) SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`insert into refresh_tokens (token, user_id, expires_at) values ($1, $2, $3)`,
		token, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("unable to save refresh token: %w", err)
	}
	return nil
}

func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{Token: token}
	err := r.db.QueryRow(ctx,
		`select user_id, expires_at, revoked, created_at from refresh_tokens where token=$1`, token,
	).Scan(&rt.UserID, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, fmt.Errorf("unable to fetch refresh token: %w", err)
	}
	return rt, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `update refresh_tokens set revoked=true where token=$1`, token)
	if err != nil {
		return fmt.Errorf("unable to revoke refresh token: %w", err)
	}
	return nil
}

// Idempotency

func (r *Repository) GetIdempotencyRecord(ctx context.Context, key string) (*models.IdempotencyRecord, error) {
	var record models.IdempotencyRecord
	var bodyJSON []byte
	err := r.db.QueryRow(ctx,
		`select request_hash, status_code, body, created_at from idempotency_records
		 where key=$1 and created_at > now() - interval '30 minutes'`, key,
	).Scan(&record.RequestHash, &record.StatusCode, &bodyJSON, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found = not an error
		}
		return nil, fmt.Errorf("unable to fetch idempotency record: %w", err)
	}
	if err := json.Unmarshal(bodyJSON, &record.Body); err != nil {
		return nil, fmt.Errorf("unable to unmarshal idempotency body: %w", err)
	}
	return &record, nil
}

func (r *Repository) SaveIdempotencyRecord(ctx context.Context, key, requestHash string, statusCode int, body any) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("unable to marshal idempotency body: %w", err)
	}
	_, err = r.db.Exec(ctx,
		`insert into idempotency_records (key, request_hash, status_code, body) values ($1, $2, $3, $4)
		 on conflict (key) do update set request_hash=excluded.request_hash,
		   status_code=excluded.status_code, body=excluded.body, created_at=now()`,
		key, requestHash, statusCode, bodyJSON)
	if err != nil {
		return fmt.Errorf("unable to save idempotency record: %w", err)
	}
	return nil
}
