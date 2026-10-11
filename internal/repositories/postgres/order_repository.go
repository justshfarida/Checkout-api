package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresRepository) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to begin transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	// Lock all requested items and verify stock.
	for _, lineItem := range items {
		var item models.Item

		err := tx.QueryRow(
			ctx,
			`
			SELECT
				id,
				name,
				description,
				price,
				stock,
				created_at,
				item_type
			FROM items
			WHERE id = $1
			FOR UPDATE
			`,
			lineItem.ItemID,
		).Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Stock,
			&item.CreatedAt,
			&item.ItemType,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
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

		// Only physical merchandise has limited stock.
		// Virtual items and currency packages have unlimited stock.
		if item.ItemType == "merch" &&
			item.Stock < lineItem.Quantity {

			return nil, fmt.Errorf(
				"insufficient stock for item %d: have %d, need %d",
				lineItem.ItemID,
				item.Stock,
				lineItem.Quantity,
			)
		}
	}

	var orderID int
	var createdAt time.Time

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			total,
			status
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
		`,
		userID,
		total,
		status,
	).Scan(
		&orderID,
		&createdAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to insert order: %w",
			err,
		)
	}

	for _, lineItem := range items {
		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO line_items (
				order_id,
				item_id,
				price,
				quantity
			)
			VALUES ($1, $2, $3, $4)
			`,
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
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"failed to commit transaction: %w",
			err,
		)
	}

	return &models.Order{
		ID:        orderID,
		UserID:    userID,
		Items:     items,
		Total:     total,
		Status:    status,
		CreatedAt: createdAt,
	}, nil
}

func (r *PostgresRepository) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	var (
		result pgconn.CommandTag
		err    error
	)

	if status == "canceled" {
		result, err = r.pool.Exec(
			ctx,
			`
			UPDATE orders
			SET status = $1
			WHERE id = $2
			  AND status = 'pending'
			`,
			status,
			orderID,
		)
	} else {
		result, err = r.pool.Exec(
			ctx,
			`
			UPDATE orders
			SET status = $1
			WHERE id = $2
			`,
			status,
			orderID,
		)
	}

	if err != nil {
		return fmt.Errorf(
			"failed to update order status: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return repositories.ErrOrderNotFound
	}

	return nil
}

func (r *PostgresRepository) GetOrdersByUserIDOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.UserOrder, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		WITH paginated_orders AS (
			SELECT
				id,
				user_id,
				total,
				status
			FROM orders
			WHERE user_id = $1
			ORDER BY id DESC
			LIMIT $2
			OFFSET $3
		)
		SELECT
			o.id,
			o.user_id,
			o.total,
			o.status,
			li.item_id,
			i.name,
			li.price,
			li.quantity
		FROM paginated_orders o
		LEFT JOIN line_items li
			ON li.order_id = o.id
		LEFT JOIN items i
			ON i.id = li.item_id
		ORDER BY o.id DESC, li.item_id
		`,
		userID,
		limit,
		offset,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get orders: %w",
			err,
		)
	}

	defer rows.Close()

	return scanOrders(rows)
}

func (r *PostgresRepository) GetOrdersByUserIDCursor(
	ctx context.Context,
	userID int,
	lastID int,
	limit int,
) ([]models.UserOrder, error) {
	var rows pgx.Rows
	var err error

	if lastID == 0 {
		// First page: no cursor yet.
		rows, err = r.pool.Query(
			ctx,
			`
			WITH paginated_orders AS (
				SELECT
					id,
					user_id,
					total,
					status
				FROM orders
				WHERE user_id = $1
				ORDER BY id DESC
				LIMIT $2
			)
			SELECT
				o.id,
				o.user_id,
				o.total,
				o.status,
				li.item_id,
				i.name,
				li.price,
				li.quantity
			FROM paginated_orders o
			LEFT JOIN line_items li
				ON li.order_id = o.id
			LEFT JOIN items i
				ON i.id = li.item_id
			ORDER BY o.id DESC, li.item_id
			`,
			userID,
			limit,
		)
	} else {
		rows, err = r.pool.Query(
			ctx,
			`
			WITH paginated_orders AS (
				SELECT
					id,
					user_id,
					total,
					status
				FROM orders
				WHERE user_id = $1
				  AND id < $2
				ORDER BY id DESC
				LIMIT $3
			)
			SELECT
				o.id,
				o.user_id,
				o.total,
				o.status,
				li.item_id,
				i.name,
				li.price,
				li.quantity
			FROM paginated_orders o
			LEFT JOIN line_items li
				ON li.order_id = o.id
			LEFT JOIN items i
				ON i.id = li.item_id
			ORDER BY o.id DESC, li.item_id
			`,
			userID,
			lastID,
			limit,
		)
	}

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get orders: %w",
			err,
		)
	}

	defer rows.Close()

	return scanOrders(rows)
}
func (r *PostgresRepository) MarkOrderPaid(
	ctx context.Context,
	orderID int,
	items []models.LineItem,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	for _, item := range items {
		var itemType string

		err := tx.QueryRow(
			ctx,
			`
			SELECT item_type
			FROM items
			WHERE id = $1
			`,
			item.ItemID,
		).Scan(&itemType)

		if err != nil {
			return err
		}

		// Only physical merchandise has limited stock.
		// Virtual items and currency packages have unlimited stock.
		if itemType != "merch" {
			continue
		}

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
			return err
		}

		if result.RowsAffected() == 0 {
			return validation.ErrInsufficientStock
		}
	}

	_, err = tx.Exec(
		ctx,
		`
		UPDATE orders
		SET status = 'paid'
		WHERE id = $1
		`,
		orderID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func (r *PostgresRepository) GetOrderItems(
	ctx context.Context,
	orderID int,
) ([]models.LineItem, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			item_id,
			quantity,
			price
		FROM line_items
		WHERE order_id = $1
		ORDER BY item_id
		`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get order items: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]models.LineItem, 0)

	for rows.Next() {
		var item models.LineItem

		if err := rows.Scan(
			&item.ItemID,
			&item.Quantity,
			&item.Price,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to scan order item: %w",
				err,
			)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed while reading order items: %w",
			err,
		)
	}

	return items, nil
}
func scanOrders(
	rows pgx.Rows,
) ([]models.UserOrder, error) {
	orders := make(
		[]models.UserOrder,
		0,
	)

	orderIndexes := make(
		map[int]int,
	)

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
				"failed to scan order: %w",
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

			index = len(orders) - 1

			orderIndexes[orderID] = index
		}

		if itemID != nil {
			orderItem :=
				models.OrderItem{
					ItemID: *itemID,
				}

			if itemName != nil {
				orderItem.Name = *itemName
			}

			if price != nil {
				orderItem.Price = *price
			}

			if quantity != nil {
				orderItem.Quantity = *quantity
			}

			orders[index].Items = append(
				orders[index].Items,
				orderItem,
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

// GetOrderForUser loads one order with its items. Ownership is part of the
// WHERE clause (id AND user_id), not a check added afterwards: the database
// simply never returns another user's order, so "not found" and "not yours"
// look identical and nobody can probe which order IDs exist.
func (r *PostgresRepository) GetOrderForUser(
	ctx context.Context,
	orderID int,
	userID int,
) (*models.UserOrder, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			o.id,
			o.user_id,
			o.total,
			o.status,
			li.item_id,
			i.name,
			li.price,
			li.quantity
		FROM orders o
		LEFT JOIN line_items li
			ON li.order_id = o.id
		LEFT JOIN items i
			ON i.id = li.item_id
		WHERE o.id = $1 AND o.user_id = $2
		ORDER BY li.item_id
		`,
		orderID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	defer rows.Close()

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, repositories.ErrOrderNotFound
	}
	return &orders[0], nil
}
