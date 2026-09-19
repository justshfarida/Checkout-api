package services

import (
	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"
	"context"
	"fmt"
)

type PaymentResult struct {
	Success       bool
	TransactionID string
	Error         string
}

type CreateOrderResult struct {
	Order *models.Order
	// Payment       PaymentResult
	XsollaOrderID int    `json:"xsolla_order_id"`
	PaymentURL    string `json:"payment_url"`
}
type XsollaPaymentService interface {
	CreatePayment(
		ctx context.Context,
		userID int,
		orderID int,
		items []XsollaItem,
	) (*XsollaPaymentResponse, error)
}
type OrderService struct {
	repo          repositories.OrderRepository
	itemRepo      repositories.ItemRepository
	cartRepo      repositories.CartRepository
	payment       PaymentProcessor
	xsollaService XsollaPaymentService
	xsollaRep repositories.XsollaRepository
}

func NewOrderService(
	repo repositories.OrderRepository,
	cartRepo repositories.CartRepository,
	itemRepo repositories.ItemRepository,
	payment PaymentProcessor,
	xsollaService XsollaPaymentService,
	xsollaRep repositories.XsollaRepository, 
) *OrderService {
	return &OrderService{
		repo:          repo,
		cartRepo:      cartRepo,
		itemRepo:      itemRepo,
		payment:       payment,
		xsollaService: xsollaService,
		xsollaRep: xsollaRep,
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
	xsollaItems := make(
		[]XsollaItem,
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
		xsollaItems = append(
			xsollaItems,
			XsollaItem{
				SKU:      fmt.Sprintf("item_%d", item.ID),
				Quantity: cartItem.Quantity,
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

	payment, err := s.xsollaService.CreatePayment(
		ctx,
		userID,
		order.ID,
		xsollaItems,
	)
	if err != nil {
		return nil, err
	}

	paymentURL :=
		"https://sandbox-secure.xsolla.com/paystation4/?token=" +
			payment.Token

	return &CreateOrderResult{
		Order:         order,
		XsollaOrderID: payment.OrderID,
		PaymentURL:    paymentURL,
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
func (s *OrderService) ProcessXsollaPayment(
	ctx context.Context,
	orderID int,
	transactionID int64,
) error {
	items, err := s.repo.GetOrderItems(
		ctx,
		orderID,
	)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return ErrOrderNotFound
	}

	_, err = s.xsollaRep.ProcessXsollaPayment(
		ctx,
		transactionID,
		orderID,
		items,
	)
	if err != nil {
		return err
	}

	return nil
}