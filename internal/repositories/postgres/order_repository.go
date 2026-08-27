package postgres

import (
	"context"
	"errors"
	"fmt"

	"checkout-api/internal/models"

	"github.com/jackc/pgx/v5"
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
				created_at
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

		if item.Stock < lineItem.Quantity {
			return nil, fmt.Errorf(
				"insufficient stock for item %d: have %d, need %d",
				lineItem.ItemID,
				item.Stock,
				lineItem.Quantity,
			)
		}
	}

	var orderID int

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			total,
			status
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		userID,
		total,
		status,
	).Scan(&orderID)

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

		_, err = tx.Exec(
			ctx,
			`
			UPDATE items
			SET stock = stock - $1
			WHERE id = $2
			`,
			lineItem.Quantity,
			lineItem.ItemID,
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

func (r *PostgresRepository) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		UPDATE orders
		SET status = $1
		WHERE id = $2
		`,
		status,
		orderID,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to update order status: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresRepository) GetOrdersByUserID(
	ctx context.Context,
	userID int,
) ([]models.UserOrder, error) {
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
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC, li.id
		`,
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

	orders := make([]models.UserOrder, 0)
	orderIndexes := make(map[int]int)

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

		if err := rows.Scan(
			&orderID,
			&orderUserID,
			&total,
			&status,
			&itemID,
			&itemName,
			&price,
			&quantity,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to scan order row: %w",
				err,
			)
		}

		index, exists := orderIndexes[orderID]

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
			orderItem := models.OrderItem{
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
