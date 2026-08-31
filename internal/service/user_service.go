package service

import (
	"checkout-api/internal/models"
	"checkout-api/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	SignUp(ctx context.Context, username string, passwordHash string) (*models.User, error)
	Login(ctx context.Context, username string) (*models.User, error)
}

type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) SignUp(ctx context.Context, username string, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.repo.SignUp(ctx, username, string(hashedPassword))
	if err != nil {
		if errors.Is(err, repository.ErrUsernameTaken) {
			return nil, fmt.Errorf("%w: %q", ErrUsernameTaken, username)
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, username, password string) (*models.User, error) {
	user, err := s.repo.Login(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

type AuthManager interface {
	GenerateAccessToken(userID int) (string, error)
	GenerateRefreshToken() (string, error)
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type RefreshTokenService struct {
	repo        RefreshTokenRepository
	authManager AuthManager
}

func NewRefreshTokenService(repo RefreshTokenRepository, authManager AuthManager) *RefreshTokenService {
	return &RefreshTokenService{repo: repo, authManager: authManager}
}

func (s *RefreshTokenService) SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error {
	if err := s.repo.SaveRefreshToken(ctx, token, userID, expiresAt); err != nil {
		return fmt.Errorf("failed to save refresh token for user %d: %w", userID, err)
	}
	return nil
}

func (s *RefreshTokenService) Rotate(ctx context.Context, oldToken string) (*TokenPair, error) {
	rt, err := s.repo.GetRefreshToken(ctx, oldToken)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to fetch refresh token: %w", err)
	}
	if rt.Revoked {
		return nil, ErrRefreshTokenRevoked
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}

	if err := s.repo.RevokeRefreshToken(ctx, oldToken); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	newAccessToken, err := s.authManager.GenerateAccessToken(rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	newRefreshToken, err := s.authManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	if err := s.repo.SaveRefreshToken(ctx, newRefreshToken, rt.UserID, time.Now().Add(30*24*time.Hour)); err != nil {
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}

	return &TokenPair{AccessToken: newAccessToken, RefreshToken: newRefreshToken}, nil
}
