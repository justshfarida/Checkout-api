package service

import (
	pagination2 "checkout-api/internal/controllers/pagination"
	"checkout-api/internal/models"
	"checkout-api/internal/repository"
	"context"
	"errors"
	"fmt"
)

type ItemsRepository interface {
	ListOffset(ctx context.Context, filter models.ItemFilter, page pagination2.OffsetPage) ([]*models.Item, int, error)
	ListCursor(ctx context.Context, filter models.ItemFilter, page pagination2.CursorPage) ([]*models.Item, bool, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
}

type ItemsService struct {
	repo ItemsRepository
}

func NewItemsService(repo ItemsRepository) *ItemsService {
	return &ItemsService{repo: repo}
}

func (s *ItemsService) ListOffset(ctx context.Context, filter models.ItemFilter, page pagination2.OffsetPage) (*pagination2.PageResult[*models.Item], error) {
	items, total, err := s.repo.ListOffset(ctx, filter, page)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	return &pagination2.PageResult[*models.Item]{
		Items: items, Total: total, Limit: page.Limit, Offset: page.Offset,
	}, nil
}

func (s *ItemsService) ListCursor(ctx context.Context, filter models.ItemFilter, page pagination2.CursorPage) (*pagination2.PageResult[*models.Item], error) {
	items, hasMore, err := s.repo.ListCursor(ctx, filter, page)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	result := &pagination2.PageResult[*models.Item]{Items: items, Limit: page.Limit, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		result.NextCursor = pagination2.EncodeCursor(items[len(items)-1].ID)
	}
	return result, nil
}

func (s *ItemsService) GetItem(ctx context.Context, id int) (*models.Item, error) {
	item, err := s.repo.GetItem(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("%w: item %d", ErrItemNotFound, id)
		}
		return nil, fmt.Errorf("failed to fetch item %d: %w", id, err)
	}
	return item, nil
}
