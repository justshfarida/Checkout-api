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
	cartRepo repositories.CartRepository
}

func NewOrderService(
	repo repositories.OrderRepository,
	cartRepo repositories.CartRepository,
	itemRepo repositories.ItemRepository,
) *OrderService {
	return &OrderService{
		repo:     repo,
		cartRepo: cartRepo,
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
) (*CreateOrderResult, error) {
	cartItems, err := s.cartRepo.GetUserCartOffset(
		ctx,
		userID,
		1000,
		0,
	)
	if err != nil {
		return nil, err
	}

	if len(cartItems) == 0 {
		return nil, ErrEmptyCart
	}

	lineItems := make(
		[]models.LineItem,
		0,
		len(cartItems),
	)

	total := 0

	for _, cartItem := range cartItems {
		item, err := s.itemRepo.GetItem(
			ctx,
			cartItem.ID,
		)
		if err != nil {
			return nil, err
		}

		if item == nil {
			return nil, ErrItemNotFound
		}

		if err := validation.ValidateStock(
			item.Stock,
			cartItem.Quantity,
		); err != nil {
			return nil, err
		}

		lineItems = append(
			lineItems,
			models.LineItem{
				ItemID:   item.ID,
				Quantity: cartItem.Quantity,
				Price:    item.Price,
			},
		)

		total += item.Price * cartItem.Quantity
	}

	order, err := s.repo.CreateOrder(
		ctx,
		userID,
		lineItems,
		total,
		"pending",
	)
	if err != nil {
		return nil, err
	}

	payment := mockProcessPayment(total)

	if !payment.Success {
		if err := s.repo.UpdateOrderStatus(ctx, order.ID, "failed"); err != nil {
			return nil, err
		}
		order.Status = "failed"
		return &CreateOrderResult{
			Order:   order,
			Payment: payment,
		}, nil
	}

	if err := s.repo.MarkOrderPaid(
		ctx,
		order.ID,
		lineItems,
	); err != nil {
		return nil, err
	}

	order.Status = "paid"
	if err := s.cartRepo.DeleteUserCart(ctx, userID); err != nil {
		return nil, err
	}
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
