package services

import (
	"context"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

type ItemService struct {
	repo repositories.ItemRepository
}

func NewItemService(
	repo repositories.ItemRepository,
) *ItemService {
	return &ItemService{
		repo: repo,
	}
}

func (s *ItemService) GetItems(
	ctx context.Context,
) ([]*models.Item, error) {
	return s.repo.GetItems(ctx)
}

func (s *ItemService) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	return s.repo.GetItem(ctx, id)
}
