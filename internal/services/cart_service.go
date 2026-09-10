package services

import (
	"context"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"
)

type CartWriter interface {
	UpsertCartItem(ctx context.Context, userID int, itemID int, quantity int) error
	DeleteUserCart(ctx context.Context, userID int) error
	RemoveCartItem(ctx context.Context, userID int, itemID int) error
}

type CartService struct {
	repo     repositories.CartRepository
	writer   CartWriter
	itemRepo repositories.ItemRepository
}

func NewCartService(
	repo repositories.CartRepository,
	writer CartWriter,
	itemRepo repositories.ItemRepository,
) *CartService {
	return &CartService{
		repo:     repo,
		writer:   writer,
		itemRepo: itemRepo,
	}
}

type CartOffsetResult struct {
	Items  []models.CartItem `json:"items"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Total  int               `json:"total"`
}

type CartCursorResult struct {
	Items      []models.CartItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
	Total      int               `json:"total"`
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

	total, err := s.repo.GetUserCartTotal(
		ctx,
		userID,
	)
	if err != nil {
		return CartOffsetResult{}, err
	}

	return CartOffsetResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Total:  total,
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
		nextCursor = EncodeCursor(
			items[len(items)-1].ID,
		)
	}

	total, err := s.repo.GetUserCartTotal(
		ctx,
		userID,
	)
	if err != nil {
		return CartCursorResult{}, err
	}

	return CartCursorResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      total,
	}, nil
}

func (s *CartService) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	item, err := s.itemRepo.GetItem(
		ctx,
		itemID,
	)
	if err != nil {
		return err
	}

	if item == nil {
		return ErrItemNotFound
	}

	if err := validation.ValidateStock(
		item.Stock,
		quantity,
	); err != nil {
		return err
	}

	return s.writer.UpsertCartItem(
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
	return s.writer.DeleteUserCart(
		ctx,
		userID,
	)
}

func (s *CartService) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	return s.writer.RemoveCartItem(
		ctx,
		userID,
		itemID,
	)
}
