package handlers

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/models"
	"context"
	"time"
)

type Store interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)

	CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
	UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error)
	RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error)

	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error)

	SignUp(ctx context.Context, username string, password string) (*models.User, error)
	Login(ctx context.Context, username, password string) (*models.User, error)
	SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error

	SaveIdempotencyRecord(ctx context.Context, key, requestHash string, statusCode int, body any) error
	GetIdempotencyRecord(ctx context.Context, key string) (*models.IdempotencyRecord, error)
}

type Handler struct {
	store       Store
	authManager *auth.Manager
}

func NewHandler(s Store, authManager *auth.Manager) *Handler {
	return &Handler{store: s, authManager: authManager}
}
