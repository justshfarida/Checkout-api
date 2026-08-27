package services

import (
	"context"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

type AuthService struct {
	repo repositories.UserRepository
}

func NewAuthService(
	repo repositories.UserRepository,
) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

func (s *AuthService) FindUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	return s.repo.FindUserByEmail(
		ctx,
		email,
	)
}

func (s *AuthService) SaveUser(
	ctx context.Context,
	email string,
	hash []byte,
) error {
	return s.repo.SaveUser(
		ctx,
		email,
		hash,
	)
}

func (s *AuthService) SaveRefreshToken(
	ctx context.Context,
	userID int,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	return s.repo.SaveRefreshToken(
		ctx,
		userID,
		tokenHash,
		expiresAt,
	)
}

func (s *AuthService) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash []byte,
	newTokenHash []byte,
	newExpiresAt time.Time,
) (int, error) {
	return s.repo.RotateRefreshToken(
		ctx,
		oldTokenHash,
		newTokenHash,
		newExpiresAt,
	)
}
