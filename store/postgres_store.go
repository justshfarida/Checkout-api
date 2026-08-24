package store

import (
	"checkout-api/models"

	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore stores PostgreSQL-related dependencies.
type PostgresStore struct {
	query *Query
	pool  *pgxpool.Pool
}

// NewPostgresStore creates a PostgreSQL store
// backed by a connection pool.
func NewPostgresStore(
	pool *pgxpool.Pool,
) *PostgresStore {
	return &PostgresStore{
		query: &Query{
			DBTX: pool,
		},
		pool: pool,
	}
}

// DB returns the normal query object
// backed by the PostgreSQL connection pool.
func (s *PostgresStore) DB() *Query {
	return s.query
}

// WithTx creates a Query that executes
// inside the supplied transaction.
func (s *PostgresStore) WithTx(
	tx pgx.Tx,
) *Query {
	return &Query{
		DBTX: tx,
	}
}

// --------------------
// ITEMS
// --------------------

func (s *PostgresStore) GetItems(
	ctx context.Context,
) ([]*models.Item, error) {
	rows, err :=
		s.DB().GetItems(ctx)

	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to select all items",
			err,
		)
	}

	defer rows.Close()

	items := make(
		[]*models.Item,
		0,
	)

	for rows.Next() {
		var item models.Item

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"unable to scan row: %w",
				err,
			)
		}

		items = append(
			items,
			&item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *PostgresStore) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	var item models.Item

	err :=
		s.DB().
			GetItemByID(
				ctx,
				id,
			).
			Scan(
				&item.ID,
				&item.Name,
				&item.Description,
				&item.Price,
				&item.Stock,
				&item.CreatedAt,
			)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil, nil
		}

		return nil, err
	}

	return &item, nil
}

// --------------------
// ORDERS
// --------------------

func (s *PostgresStore) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	tx, err :=
		s.pool.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to begin transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	q := s.WithTx(tx)

	// Lock items and verify stock.
	for _, lineItem := range items {
		var item models.Item

		err :=
			q.GetItemByIDForUpdate(
				ctx,
				lineItem.ItemID,
			).
				Scan(
					&item.ID,
					&item.Name,
					&item.Description,
					&item.Price,
					&item.Stock,
					&item.CreatedAt,
				)

		if err != nil {
			if errors.Is(
				err,
				pgx.ErrNoRows,
			) {
				return nil, fmt.Errorf(
					"item %d not found",
					lineItem.ItemID,
				)
			}

			return nil, fmt.Errorf(
				"failed to lock item %d: %w",
				lineItem.ItemID,
				err,
			)
		}

		if item.Stock <
			lineItem.Quantity {

			return nil, fmt.Errorf(
				"insufficient stock for item %d: have %d, need %d",
				lineItem.ItemID,
				item.Stock,
				lineItem.Quantity,
			)
		}
	}

	var orderID int

	err =
		q.InsertOrderReturning(
			ctx,
			userID,
			total,
			status,
		).
			Scan(
				&orderID,
			)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to insert order: %w",
			err,
		)
	}

	for _, lineItem := range items {
		_, err :=
			q.InsertLineItem(
				ctx,
				orderID,
				lineItem.ItemID,
				lineItem.Price,
				lineItem.Quantity,
			)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to insert line item for item %d: %w",
				lineItem.ItemID,
				err,
			)
		}

		_, err =
			q.DecrementItemStock(
				ctx,
				lineItem.ItemID,
				lineItem.Quantity,
			)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to decrement stock for item %d: %w",
				lineItem.ItemID,
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"failed to commit transaction: %w",
			err,
		)
	}

	return &models.Order{
		ID:     orderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}, nil
}

func (s *PostgresStore) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	_, err :=
		s.DB().
			UpdateOrderStatus(
				ctx,
				orderID,
				status,
			)

	return err
}

func (s *PostgresStore) GetOrdersByUserID(
	ctx context.Context,
	userID int,
) ([]models.UserOrder, error) {
	rows, err :=
		s.DB().GetOrdersByUserID(
			ctx,
			userID,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get orders for user %d: %w",
			userID,
			err,
		)
	}

	defer rows.Close()

	orders := make(
		[]models.UserOrder,
		0,
	)

	orderIndexes :=
		make(map[int]int)

	for rows.Next() {
		var (
			orderID     int
			orderUserID int
			total       int
			status      string

			itemID   *int
			itemName *string
			price    *int
			quantity *int
		)

		err := rows.Scan(
			&orderID,
			&orderUserID,
			&total,
			&status,
			&itemID,
			&itemName,
			&price,
			&quantity,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan order row: %w",
				err,
			)
		}

		index, exists :=
			orderIndexes[orderID]

		if !exists {
			orders = append(
				orders,
				models.UserOrder{
					ID:     orderID,
					UserID: orderUserID,
					Total:  total,
					Status: status,
					Items:  []models.OrderItem{},
				},
			)

			index =
				len(orders) - 1

			orderIndexes[orderID] =
				index
		}

		if itemID != nil {
			orders[index].Items =
				append(
					orders[index].Items,
					models.OrderItem{
						ItemID: *itemID,
						Name: func() string {
							if itemName == nil {
								return ""
							}

							return *itemName
						}(),
						Price: func() int {
							if price == nil {
								return 0
							}

							return *price
						}(),
						Quantity: func() int {
							if quantity == nil {
								return 0
							}

							return *quantity
						}(),
					},
				)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed while reading orders: %w",
			err,
		)
	}

	return orders, nil
}

// --------------------
// CART
// --------------------

func (s *PostgresStore) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	_, err :=
		s.DB().
			UpsertCart(
				ctx,
				userID,
				itemID,
				quantity,
			)

	return err
}

func (s *PostgresStore) GetUserCart(
	ctx context.Context,
	userID int,
) ([]models.CartItem, error) {
	rows, err :=
		s.DB().
			GetItemsFromUserCart(
				ctx,
				userID,
			)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get cart for user %d: %w",
			userID,
			err,
		)
	}

	defer rows.Close()

	items := make(
		[]models.CartItem,
		0,
	)

	for rows.Next() {
		var item models.CartItem

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
			&item.Quantity,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan cart row: %w",
				err,
			)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *PostgresStore) DeleteUserCart(
	ctx context.Context,
	userID int,
) error {
	_, err :=
		s.DB().
			DeleteCartByUserID(
				ctx,
				userID,
			)

	return err
}

func (s *PostgresStore) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	_, err :=
		s.DB().
			DeleteItemFromUserCart(
				ctx,
				userID,
				itemID,
			)

	return err
}

// --------------------
// USERS
// --------------------

func (s *PostgresStore) FindUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	var user models.User

	row :=
		s.DB().
			GetUserByEmail(
				ctx,
				email,
			)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Hash,
	)

	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *PostgresStore) SaveUser(
	ctx context.Context,
	email string,
	hash []byte,
) error {
	_, err :=
		s.DB().
			InsertUser(
				ctx,
				email,
				hash,
			)

	return err
}

// --------------------
// REFRESH TOKENS
// --------------------

func (s *PostgresStore) SaveRefreshToken(
	ctx context.Context,
	userID int,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	_, err :=
		s.DB().
			InsertRefreshToken(
				ctx,
				userID,
				tokenHash,
				expiresAt,
			)

	if err != nil {
		return fmt.Errorf(
			"failed to save refresh token: %w",
			err,
		)
	}

	return nil
}

func (s *PostgresStore) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash []byte,
	newTokenHash []byte,
	newExpiresAt time.Time,
) (int, error) {
	tx, err :=
		s.pool.Begin(ctx)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to begin refresh-token transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	q := s.WithTx(tx)

	var userID int
	var oldExpiresAt time.Time

	err =
		q.GetActiveRefreshTokenForUpdate(
			ctx,
			oldTokenHash,
		).
			Scan(
				&userID,
				&oldExpiresAt,
			)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return 0, fmt.Errorf(
				"invalid or inactive refresh token",
			)
		}

		return 0, fmt.Errorf(
			"failed to retrieve refresh token: %w",
			err,
		)
	}

	if time.Now().After(
		oldExpiresAt,
	) {
		return 0, fmt.Errorf(
			"refresh token has expired",
		)
	}

	commandTag, err :=
		q.DeactivateRefreshToken(
			ctx,
			oldTokenHash,
		)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to deactivate refresh token: %w",
			err,
		)
	}

	if commandTag.RowsAffected() != 1 {
		return 0, fmt.Errorf(
			"failed to deactivate refresh token",
		)
	}

	_, err =
		q.InsertRefreshToken(
			ctx,
			userID,
			newTokenHash,
			newExpiresAt,
		)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to save new refresh token: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(
			"failed to commit refresh-token rotation: %w",
			err,
		)
	}

	return userID, nil
}
