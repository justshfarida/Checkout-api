package service

import (
	"checkout-api/internal/models"
	"context"
	"fmt"
)

const (
	StatusPaymentRequired = 402
	StatusCreated         = 201
)

type OrdersRepository interface {
	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error)
}

type IdempotencyRepository interface {
	GetIdempotencyRecord(ctx context.Context, key string) (*models.IdempotencyRecord, error)
	SaveIdempotencyRecord(ctx context.Context, key, requestHash string, statusCode int, body any) error
}

type PlaceOrderResult struct {
	StatusCode int
	Body       any
}

type OrdersService struct {
	ordersRepo      OrdersRepository
	cartRepo        CartRepository
	itemsRepo       ItemsRepository
	idempotencyRepo IdempotencyRepository
	paymentService  *PaymentService
}

func NewOrdersService(
	ordersRepo OrdersRepository,
	cartRepo CartRepository,
	itemsRepo ItemsRepository,
	idempotencyRepo IdempotencyRepository,
	paymentService *PaymentService,
) *OrdersService {
	return &OrdersService{
		ordersRepo:      ordersRepo,
		cartRepo:        cartRepo,
		itemsRepo:       itemsRepo,
		idempotencyRepo: idempotencyRepo,
		paymentService:  paymentService,
	}
}

func (s *OrdersService) PlaceOrder(ctx context.Context, userID int, idempotencyKey, requestHash string) (*PlaceOrderResult, error) {
	cached, err := s.idempotencyRepo.GetIdempotencyRecord(ctx, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency record: %w", err)
	}
	if cached != nil {
		if cached.RequestHash != requestHash {
			return nil, ErrIdempotencyConflict
		}
		return &PlaceOrderResult{StatusCode: cached.StatusCode, Body: cached.Body}, nil
	}

	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cart: %w", err)
	}
	if cart == nil || len(cart.Items) == 0 {
		return nil, ErrCartEmpty
	}

	total := 0
	orderItems := make([]models.LineItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("%w: item %d", ErrInvalidQuantity, item.ItemID)
		}
		product, err := s.itemsRepo.GetItem(ctx, item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("%w: item %d", ErrItemNotFound, item.ItemID)
		}
		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("%w: item %d", ErrInsufficientStock, item.ItemID)
		}
		total += product.Price * item.Quantity
		orderItems = append(orderItems, models.LineItem{
			ItemID: item.ItemID, Name: product.Name, Quantity: item.Quantity, Price: product.Price,
		})
	}

	paymentResult := s.paymentService.ProcessPayment(total)
	if !paymentResult.Success {
		body := map[string]any{"payment": paymentResult}
		if err := s.idempotencyRepo.SaveIdempotencyRecord(ctx, idempotencyKey, requestHash, StatusPaymentRequired, body); err != nil {
			return nil, fmt.Errorf("failed to save idempotency record: %w", err)
		}
		return &PlaceOrderResult{StatusCode: StatusPaymentRequired, Body: body}, nil
	}

	order, err := s.ordersRepo.CreateOrder(ctx, userID, orderItems, total, "paid")
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	if err := s.cartRepo.DeleteUserCart(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to clear cart: %w", err)
	}

	body := map[string]any{"order": order, "payment": paymentResult}
	if err := s.idempotencyRepo.SaveIdempotencyRecord(ctx, idempotencyKey, requestHash, StatusCreated, body); err != nil {
		return nil, fmt.Errorf("failed to save idempotency record: %w", err)
	}
	return &PlaceOrderResult{StatusCode: StatusCreated, Body: body}, nil
}

func (s *OrdersService) GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error) {
	orders, err := s.ordersRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orders for user %d: %w", userID, err)
	}
	return orders, nil
}
