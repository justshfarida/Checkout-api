package services

import (
	"context"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

type CartService struct {
	repo repositories.CartRepository
}

func NewCartService(
	repo repositories.CartRepository,
) *CartService {
	return &CartService{
		repo: repo,
	}
}

func (s *CartService) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	return s.repo.UpsertCartItem(
		ctx,
		userID,
		itemID,
		quantity,
	)
}

func (s *CartService) GetUserCart(
	ctx context.Context,
	userID int,
) ([]models.CartItem, error) {
	return s.repo.GetUserCart(
		ctx,
		userID,
	)
}

func (s *CartService) DeleteUserCart(
	ctx context.Context,
	userID int,
) error {
	return s.repo.DeleteUserCart(
		ctx,
		userID,
	)
}

func (s *CartService) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	return s.repo.RemoveCartItem(
		ctx,
		userID,
		itemID,
	)
}
