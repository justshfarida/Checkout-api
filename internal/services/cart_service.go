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

type CartOffsetResult struct {
	Items  []models.CartItem `json:"items"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

type CartCursorResult struct {
	Items      []models.CartItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

func (s *CartService) GetUserCartOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) (CartOffsetResult, error) {
	limit = NormalizeLimit(limit)

	if offset < 0 {
		offset = 0
	}

	items, err := s.repo.GetUserCartOffset(
		ctx,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return CartOffsetResult{}, err
	}

	return CartOffsetResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *CartService) GetUserCartCursor(
	ctx context.Context,
	userID int,
	rawCursor string,
	limit int,
) (CartCursorResult, error) {
	limit = NormalizeLimit(limit)

	lastID, err := DecodeCursor(rawCursor)
	if err != nil {
		return CartCursorResult{}, err
	}

	// Ask for one extra row so we know
	// whether another page exists.
	items, err := s.repo.GetUserCartCursor(
		ctx,
		userID,
		lastID,
		limit+1,
	)
	if err != nil {
		return CartCursorResult{}, err
	}

	hasMore := len(items) > limit

	if hasMore {
		items = items[:limit]
	}

	var nextCursor string

	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]

		nextCursor = EncodeCursor(
			lastItem.ID,
		)
	}

	return CartCursorResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
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
