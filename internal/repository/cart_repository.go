package repository

import (
	"checkout-api/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CartRepo struct {
	db *pgxpool.Pool
}

func NewCartRepo(db *pgxpool.Pool) *CartRepo {
	r := &CartRepo{
		db: db,
	}
	return r
}

func (r *CartRepo) CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error {
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

func (r *CartRepo) GetUserCart(ctx context.Context, userID int) (*models.Cart, error) {
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

func (r *CartRepo) DeleteUserCart(ctx context.Context, userID int) error {
	_, err := r.db.Exec(ctx, "delete from carts where user_id=$1", userID)
	if err != nil {
		fmt.Printf("unable to delete user cart %d: %w", userID, err)
	}

	return nil
}

func (r *CartRepo) UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`update cart_items set quantity=$1
		 where item_id=$2 and cart_id = (select id from carts where user_id=$3)`,
		quantity, itemID, userID)
	if err != nil {
		return false, fmt.Errorf("unable to update cart item %d for user %d: %w", itemID, userID, err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *CartRepo) RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`delete from cart_items
		 where item_id=$1 and cart_id = (select id from carts where user_id=$2)`,
		itemID, userID)
	if err != nil {
		return false, fmt.Errorf("unable to remove cart item %d for user %d: %w", itemID, userID, err)
	}
	return tag.RowsAffected() > 0, nil
}
