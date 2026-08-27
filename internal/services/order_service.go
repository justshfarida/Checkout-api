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

func (s *OrderService) GetOrdersByUserID(
	ctx context.Context,
	userID int,
) ([]models.UserOrder, error) {
	return s.repo.GetOrdersByUserID(
		ctx,
		userID,
	)
}
