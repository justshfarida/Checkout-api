package service

import (
	"checkout-api/internal/models"
	"context"
	"sort"
)

type ItemsRepository interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
}
type ItemsService struct{ repo ItemsRepository }

func NewItemsService(repo ItemsRepository) *ItemsService {
	return &ItemsService{
		repo: repo}
}

func (s *ItemsService) GetItems(ctx context.Context, category, sortBy string) ([]*models.Item, error) {
	items, err := s.repo.GetItems(ctx)
	if err != nil {
		return nil, err
	}

	if category != "" {
		filteredItems := make([]*models.Item, 0, len(items))
		for _, item := range items {
			if item.Category == category {
				filteredItems = append(filteredItems, item)
			}
		}
		items = filteredItems
	}

	switch sortBy {
	case "price":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Price < items[j].Price
		})
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
	}
	return items, nil
}

func (s *ItemsService) GetItem(ctx context.Context, id int) (*models.Item, error) {
	return s.repo.GetItem(ctx, id)
}
