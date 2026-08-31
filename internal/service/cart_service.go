package service

import (
	"checkout-api/internal/models"
	"context"
	"fmt"
)

const maxQtyPerItem = 10

type CartRepository interface {
	CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
	UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error)
	RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error)
}

type CartItemInput struct {
	ItemID   int
	Quantity int
}

type CartService struct {
	cartRepo  CartRepository
	itemsRepo ItemsRepository
}

func NewCartService(cartRepo CartRepository, itemsRepo ItemsRepository) *CartService {
	return &CartService{cartRepo: cartRepo, itemsRepo: itemsRepo}
}

func (s *CartService) CreateUserCartAndAddItems(ctx context.Context, userID int, items []CartItemInput) (*models.Cart, error) {
	lineItems := make([]models.LineItem, 0, len(items))

	for _, item := range items {
		if item.Quantity > maxQtyPerItem {
			return nil, fmt.Errorf("%w: max %d per item (got %d)", ErrQuantityLimitExceeded, maxQtyPerItem, item.Quantity)
		}

		product, err := s.itemsRepo.GetItem(ctx, item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("%w: item %d does not exist", ErrItemNotFound, item.ItemID)
		}
		if item.Quantity > product.Stock {
			return nil, fmt.Errorf("%w: item %d", ErrInsufficientStock, item.ItemID)
		}

		lineItems = append(lineItems, models.LineItem{
			ItemID:   item.ItemID,
			Name:     product.Name,
			Quantity: item.Quantity,
			Price:    product.Price,
		})
	}

	cart := &models.Cart{
		ID:     fmt.Sprintf("cart_%d", userID),
		UserID: userID,
		Items:  lineItems,
	}

	if err := s.cartRepo.CreateUserCartAndAddItems(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}
	return cart, nil
}

func (s *CartService) GetUserCart(ctx context.Context, userID int) (*models.Cart, error) {
	return s.cartRepo.GetUserCart(ctx, userID)
}

func (s *CartService) DeleteUserCart(ctx context.Context, userID int) error {
	return s.cartRepo.DeleteUserCart(ctx, userID)
}

func (s *CartService) UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (*models.Cart, error) {
	if quantity > maxQtyPerItem {
		return nil, fmt.Errorf("%w: max %d per item (got %d)", ErrQuantityLimitExceeded, maxQtyPerItem, quantity)
	}

	product, err := s.itemsRepo.GetItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: item %d", ErrItemNotFound, itemID)
	}
	if quantity > product.Stock {
		return nil, fmt.Errorf("%w: item %d", ErrInsufficientStock, itemID)
	}

	found, err := s.cartRepo.UpdateCartItem(ctx, userID, itemID, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}
	if !found {
		return nil, ErrCartItemNotFound
	}

	return s.cartRepo.GetUserCart(ctx, userID)
}

func (s *CartService) RemoveCartItem(ctx context.Context, userID, itemID int) (*models.Cart, error) {
	found, err := s.cartRepo.RemoveCartItem(ctx, userID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to remove cart item: %w", err)
	}
	if !found {
		return nil, ErrCartItemNotFound
	}
	return s.cartRepo.GetUserCart(ctx, userID)
}
