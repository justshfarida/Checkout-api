package services

import (
	"context"
	"errors"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"

	"golang.org/x/crypto/bcrypt"
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

func (s *AuthService) SaveUser(
	ctx context.Context,
	email string,
	password []byte,
) error {
	hash, err := bcrypt.GenerateFromPassword(
		password,
		bcrypt.DefaultCost,
	)

	if err != nil {
		return ErrPasswordHashing
	}

	return s.repo.SaveUser(
		ctx,
		email,
		hash,
	)
}

func (s *AuthService) Authenticate(
	ctx context.Context,
	email string,
	password []byte,
) (models.User, error) {
	user, err := s.repo.FindUserByEmail(
		ctx,
		email,
	)

	if err != nil {
		if errors.Is(
			err,
			repositories.ErrUserNotFound,
		) {
			return models.User{}, ErrInvalidCredentials
		}

		return models.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword(
		user.Hash,
		password,
	); err != nil {
		return models.User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *AuthService) UserExists(
	ctx context.Context,
	id int,
) (bool, error) {
	_, err := s.repo.FindUserByID(
		ctx,
		id,
	)

	if err != nil {
		if errors.Is(
			err,
			repositories.ErrUserNotFound,
		) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
func (s *AuthService) GetUserByID(
	ctx context.Context,
	id int,
) (models.User, error) {
	return s.repo.FindUserByID(
		ctx,
		id,
	)
}
