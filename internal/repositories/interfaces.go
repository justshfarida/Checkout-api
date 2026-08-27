package repositories

import (
	"context"
	"time"

	"checkout-api/internal/models"
)

type ItemRepository interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
}

type CartRepository interface {
	UpsertCartItem(
		ctx context.Context,
		userID int,
		itemID int,
		quantity int,
	) error

	GetUserCart(
		ctx context.Context,
		userID int,
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

	GetOrdersByUserID(
		ctx context.Context,
		userID int,
	) ([]models.UserOrder, error)
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
