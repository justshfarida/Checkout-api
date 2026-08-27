package service

import (
	"checkout-api/internal/models"
	"context"
)

type CartRepository interface {
	CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
	UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error)
	RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error)
}

type CartService struct {
	repo CartRepository
}

func NewCartService(repo CartRepository) *CartService {
	return &CartService{repo: repo}
}

func (s *CartService) CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error {

}

func (s *CartService) GetUserCart(ctx context.Context, userID int) (*models.Cart, error) {

}

func (s *CartService) DeleteUserCart(ctx context.Context, userID int) error {

}

func (s *CartService) UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error) {

}

func (s *CartService) RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error) {

}
