package service

import (
	"checkout-api/internal/models"
	"context"
	"time"
)

type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

type RefreshTokenService struct {
	repo RefreshTokenRepository
}

func NewRefreshTokenService(repo RefreshTokenRepository) *RefreshTokenService {
	return &RefreshTokenService{repo: repo}
}
