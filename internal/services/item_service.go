package services

import (
	"context"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

type ItemService struct {
	repo repositories.ItemRepository
}
type ItemOffsetResult struct {
	Items  []*models.Item `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
}

type ItemCursorResult struct {
	Items      []*models.Item `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}

func NewItemService(
	repo repositories.ItemRepository,
) *ItemService {
	return &ItemService{
		repo: repo,
	}
}

func (s *ItemService) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	item, err := s.repo.GetItem(
		ctx,
		id,
	)

	// Real database/infrastructure problem.
	if err != nil {
		return nil, err
	}

	// Database worked, but item doesn't exist.
	if item == nil {
		return nil, ErrItemNotFound
	}

	// Success.
	return item, nil
}
func (s *ItemService) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) (ItemOffsetResult, error) {
	limit = NormalizeLimit(limit)

	if offset < 0 {
		offset = 0
	}

	items, err :=
		s.repo.GetItemsOffset(
			ctx,
			limit,
			offset,
		)

	if err != nil {
		return ItemOffsetResult{}, err
	}

	total, err :=
		s.repo.CountItems(ctx)

	if err != nil {
		return ItemOffsetResult{}, err
	}

	return ItemOffsetResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}, nil
}
func (s *ItemService) GetItemsCursor(
	ctx context.Context,
	rawCursor string,
	limit int,
) (ItemCursorResult, error) {
	limit = NormalizeLimit(limit)

	lastID, err :=
		DecodeCursor(rawCursor)

	if err != nil {
		return ItemCursorResult{}, err
	}

	items, err :=
		s.repo.GetItemsCursor(
			ctx,
			lastID,
			limit+1,
		)

	if err != nil {
		return ItemCursorResult{}, err
	}

	hasMore :=
		len(items) > limit

	if hasMore {
		items = items[:limit]
	}

	var nextCursor string

	if hasMore &&
		len(items) > 0 {

		lastItem :=
			items[len(items)-1]

		nextCursor =
			EncodeCursor(
				lastItem.ID,
			)
	}

	return ItemCursorResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
