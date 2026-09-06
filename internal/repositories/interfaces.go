package repositories

import (
	"context"
	"time"

	"checkout-api/internal/models"
)

type ItemRepository interface {
	// GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
	GetItemsOffset(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*models.Item, error)

	GetItemsCursor(
		ctx context.Context,
		lastID int,
		limit int,
	) ([]*models.Item, error)
	CountItems(
		ctx context.Context,
	) (int, error)
}

type CartRepository interface {
	UpsertCartItem(
		ctx context.Context,
		userID int,
		itemID int,
		quantity int,
	) error
	GetUserCartOffset(
		ctx context.Context,
		userID int,
		limit int,
		offset int,
	) ([]models.CartItem, error)

	GetUserCartCursor(
		ctx context.Context,
		userID int,
		lastID int,
		limit int,
	) ([]models.CartItem, error)

	DeleteUserCart(
		ctx context.Context,
		userID int,
	) error

	RemoveCartItem(
		ctx context.Context,
		userID int,
		itemID int,
	) error
	GetUserCartTotal(
		ctx context.Context,
		userID int,
	) (int, error)
}

type OrderRepository interface {
	CreateOrder(
		ctx context.Context,
		userID int,
		items []models.LineItem,
		total int,
		status string,
	) (*models.Order, error)

	UpdateOrderStatus(
		ctx context.Context,
		orderID int,
		status string,
	) error
	GetOrdersByUserIDOffset(
		ctx context.Context,
		userID int,
		limit int,
		offset int,
	) ([]models.UserOrder, error)

	GetOrdersByUserIDCursor(
		ctx context.Context,
		userID int,
		lastID int,
		limit int,
	) ([]models.UserOrder, error)
	MarkOrderPaid(
		ctx context.Context,
		orderID int,
		items []models.LineItem,
	) error
}

type UserRepository interface {
	SaveUser(
		ctx context.Context,
		email string,
		hash []byte,
	) error

	FindUserByEmail(
		ctx context.Context,
		email string,
	) (models.User, error)

	SaveRefreshToken(
		ctx context.Context,
		userID int,
		tokenHash []byte,
		expiresAt time.Time,
	) error

	RotateRefreshToken(
		ctx context.Context,
		oldTokenHash []byte,
		newTokenHash []byte,
		newExpiresAt time.Time,
	) (int, error)
}
