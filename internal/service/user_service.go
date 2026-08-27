package service

import (
	"checkout-api/internal/models"
	"context"
)

type UserRepository interface {
	SignUp(ctx context.Context, username string, passwordHash string) (*models.User, error)
	Login(ctx context.Context, username, password string) (*models.User, error)
}

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) SignUp(ctx context.Context, username string, passwordHash string) (*models.User, error) {

}

func (s *UserService) Login(ctx context.Context, username, password string) (*models.User, error) {

}
