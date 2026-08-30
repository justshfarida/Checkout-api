package services

import (
	"context"
	"fmt"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"
)

type PaymentResult struct {
	Success       bool
	TransactionID string
	Error         string
}

type CreateOrderResult struct {
	Order   *models.Order
	Payment PaymentResult
}
type OrderService struct {
	repo     repositories.OrderRepository
	itemRepo repositories.ItemRepository
}

func NewOrderService(
	repo repositories.OrderRepository,
	itemRepo repositories.ItemRepository,
) *OrderService {
	return &OrderService{
		repo:     repo,
		itemRepo: itemRepo,
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

func mockProcessPayment(
	amount int,
) PaymentResult {
	amount /= 100

	if amount > 0 && amount < 40 {
		return PaymentResult{
			Success: true,
			TransactionID: fmt.Sprintf(
				"txn_%d",
				time.Now().UnixNano(),
			),
		}
	}

	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
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
) (*CreateOrderResult, error) {

	for _, line := range items {
		item, err := s.itemRepo.GetItem(
			ctx,
			line.ItemID,
		)
		if err != nil {
			return nil, err
		}

		if item == nil {
			return nil, ErrItemNotFound
		}

		if err := validation.ValidateStock(
			item.Stock,
			line.Quantity,
		); err != nil {
			return nil, err
		}
	}

	order, err := s.repo.CreateOrder(
		ctx,
		userID,
		items,
		total,
		"pending",
	)
	if err != nil {
		return nil, err
	}

	payment := mockProcessPayment(total)

	status := "paid"
	if !payment.Success {
		status = "failed"
	}

	if err := s.repo.UpdateOrderStatus(
		ctx,
		order.ID,
		status,
	); err != nil {
		return nil, err
	}

	order.Status = status

	return &CreateOrderResult{
		Order:   order,
		Payment: payment,
	}, nil
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
