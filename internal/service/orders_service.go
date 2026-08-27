package service

import (
	"checkout-api/internal/models"
	"context"
)

type OrdersRepository interface {
	CreateOrder(ctx context.Context) ([]*models.Order, error)
	GetOrdersByUserID(ctx context.Context, id int) (*models.Order, error)
}

type OrdersService struct {
	repo OrdersRepository
}

func NewOrdersService(repo OrdersRepository) *OrdersService {
	return &OrdersService{repo: repo}
}

func CreateOrder(s *OrdersService) ([]*models.Order, error) {

}

func (s *OrdersService) GetOrdersByUserID(ctx context.Context, id int) (*models.Order, error) {
	return s.repo.GetOrdersByUserID(ctx, id)
}
