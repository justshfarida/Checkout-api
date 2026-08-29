package services

import (
	"context"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

type OrderService struct {
	repo repositories.OrderRepository
}

func NewOrderService(
	repo repositories.OrderRepository,
) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

type OrderOffsetResult struct {
	Orders []models.UserOrder `json:"orders"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type OrderCursorResult struct {
	Orders     []models.UserOrder `json:"orders"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
}

func (s *OrderService) GetOrdersByUserIDOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) (OrderOffsetResult, error) {
	limit = NormalizeLimit(limit)

	if offset < 0 {
		offset = 0
	}

	orders, err :=
		s.repo.GetOrdersByUserIDOffset(
			ctx,
			userID,
			limit,
			offset,
		)

	if err != nil {
		return OrderOffsetResult{}, err
	}

	return OrderOffsetResult{
		Orders: orders,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *OrderService) GetOrdersByUserIDCursor(
	ctx context.Context,
	userID int,
	rawCursor string,
	limit int,
) (OrderCursorResult, error) {
	limit = NormalizeLimit(limit)

	lastID, err :=
		DecodeCursor(
			rawCursor,
		)

	if err != nil {
		return OrderCursorResult{}, err
	}

	orders, err :=
		s.repo.GetOrdersByUserIDCursor(
			ctx,
			userID,
			lastID,
			limit+1,
		)

	if err != nil {
		return OrderCursorResult{}, err
	}

	hasMore :=
		len(orders) > limit

	if hasMore {
		orders =
			orders[:limit]
	}

	var nextCursor string

	if hasMore &&
		len(orders) > 0 {

		lastOrder :=
			orders[len(orders)-1]

		nextCursor =
			EncodeCursor(
				lastOrder.ID,
			)
	}

	return OrderCursorResult{
		Orders:     orders,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *OrderService) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	return s.repo.CreateOrder(
		ctx,
		userID,
		items,
		total,
		status,
	)
}

func (s *OrderService) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	return s.repo.UpdateOrderStatus(
		ctx,
		orderID,
		status,
	)
}
