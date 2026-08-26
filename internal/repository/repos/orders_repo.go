package repos

import (
	"checkout-api/internal/models"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OrdersRepo struct {
	db *pgxpool.Pool
}

func NewOrdersRepo(db *pgxpool.Pool) *OrdersRepo {
	r := &OrdersRepo{
		db: db,
	}
	return r
}

func (r *OrdersRepo) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
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

func (r *OrdersRepo) GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error) {
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
