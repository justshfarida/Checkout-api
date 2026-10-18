package services

import (
	"checkout-api/internal/messaging"
	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"

	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// orderEventPublishTimeout caps how long a webhook may wait for RabbitMQ.
// A blocked broker must never hold up the answer to Xsolla.
const orderEventPublishTimeout = 2 * time.Second

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
	xsollaRep     repositories.XsollaRepository
	events        OrderEventPublisher
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
		xsollaRep:     xsollaRep,
		events:        NoopPublisher{},
	}
}

// SetEventPublisher plugs in where order events go (RabbitMQ in production).
// Kept as a setter so the constructor and existing tests don't change.
func (s *OrderService) SetEventPublisher(p OrderEventPublisher) {
	s.events = p
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

		// Only physical merchandise has limited stock.
		// Virtual items and currency packages have unlimited stock.
		if item.ItemType == "merch" {
			if err := validation.ValidateStock(
				item.Stock,
				cartItem.Quantity,
			); err != nil {
				return nil, err
			}
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
		slog.ErrorContext(ctx, "Payment token creation failed",
			"error", err.Error(),
			"order_id", order.ID,
			"user_id", userID,
		)
		return nil, err
	}

	slog.InfoContext(ctx, "Order created",
		"order_id", order.ID,
		"user_id", userID,
		"total", total,
		"line_items", len(lineItems),
		"xsolla_order_id", payment.OrderID,
	)

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

	processed, err := s.xsollaRep.ProcessXsollaPayment(
		ctx,
		transactionID,
		orderID,
		items,
	)
	if err != nil {
		return err
	}

	// Xsolla retries webhooks; a transaction that was already handled is
	// acknowledged without changing anything, and without a second event.
	if !processed {
		slog.InfoContext(ctx, "Duplicate payment webhook ignored",
			"order_id", orderID,
			"transaction_id", transactionID,
		)
		return nil
	}

	// The payment is committed. Hand the slow follow-up work (email, later
	// analytics) to the queue instead of doing it here.
	s.publishOrderPaid(ctx, orderID, items)

	return nil
}

// publishOrderPaid announces a paid order. A failure is logged but never
// returned: the payment is already committed, and the email is not critical.
// (Exercise 2, the transactional outbox, removes the small chance of losing
// the event here.)
func (s *OrderService) publishOrderPaid(ctx context.Context, orderID int, items []models.LineItem) {
	total := 0
	for _, item := range items {
		total += item.Price * item.Quantity
	}

	ctx, cancel := context.WithTimeout(ctx, orderEventPublishTimeout)
	defer cancel()

	event := messaging.OrderPaidEvent{
		OrderID: orderID,
		Total:   total,
		PaidAt:  time.Now().UTC(),
	}

	if err := s.events.PublishOrderPaid(ctx, event); err != nil {
		slog.ErrorContext(ctx, "Failed to publish order paid event",
			"order_id", orderID,
			"error", err.Error(),
		)
		return
	}

	slog.InfoContext(ctx, "Order paid event published", "order_id", orderID)
}

// GetOrderForUser returns the caller's own order. userID always comes from
// the verified JWT, never from the request, and the repository filters by
// it, so a user can't read another user's order by changing the ID.
func (s *OrderService) GetOrderForUser(
	ctx context.Context,
	orderID int,
	userID int,
) (*models.UserOrder, error) {
	order, err := s.repo.GetOrderForUser(ctx, orderID, userID)
	if errors.Is(err, repositories.ErrOrderNotFound) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return order, nil
}
