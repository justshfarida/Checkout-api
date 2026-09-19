package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"checkout-api/internal/repositories"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 7 * 24 * time.Hour
)

type TokenService struct {
	repo      repositories.UserRepository
	jwtSecret []byte
}

func NewTokenService(
	repo repositories.UserRepository,
	jwtSecret []byte,
) *TokenService {
	return &TokenService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *TokenService) CreateAccessToken(
	userID int,
) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Subject:  strconv.Itoa(userID),
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(accessTokenDuration),
			),
		},
	)

	signedJWT, err := token.SignedString(
		s.jwtSecret,
	)

	if err != nil {
		return "",
			fmt.Errorf(
				"failed to sign access token: %w",
				err,
			)
	}

	return signedJWT, nil
}

func (s *TokenService) CreateRefreshToken(
	ctx context.Context,
	userID int,
) (
	string,
	time.Time,
	error,
) {
	refreshToken, err := generateRefreshToken()

	if err != nil {
		return "", time.Time{}, err
	}

	hash := hashRefreshToken(
		refreshToken,
	)

	expiresAt := time.Now().Add(
		refreshTokenDuration,
	)

	if err := s.repo.SaveRefreshToken(
		ctx,
		userID,
		hash,
		expiresAt,
	); err != nil {
		return "", time.Time{}, err
	}

	return refreshToken,
		expiresAt,
		nil
}

func (s *TokenService) RotateRefreshToken(
	ctx context.Context,
	oldRefreshToken string,
) (
	int,
	string,
	time.Time,
	error,
) {
	newRefreshToken, err :=
		generateRefreshToken()

	if err != nil {
		return 0,
			"",
			time.Time{},
			err
	}

	oldTokenHash :=
		hashRefreshToken(
			oldRefreshToken,
		)

	newTokenHash :=
		hashRefreshToken(
			newRefreshToken,
		)

	newExpiresAt :=
		time.Now().Add(
			refreshTokenDuration,
		)

	userID, err :=
		s.repo.RotateRefreshToken(
			ctx,
			oldTokenHash,
			newTokenHash,
			newExpiresAt,
		)

	if err != nil {
		return 0,
			"",
			time.Time{},
			err
	}

	return userID,
		newRefreshToken,
		newExpiresAt,
		nil
}

func generateRefreshToken() (
	string,
	error,
) {
	randomBytes := make(
		[]byte,
		32,
	)

	if _, err := rand.Read(
		randomBytes,
	); err != nil {
		return "",
			fmt.Errorf(
				"failed to generate refresh token: %w",
				err,
			)
	}

	return base64.
			RawURLEncoding.
			EncodeToString(
				randomBytes,
			),
		nil
}

func hashRefreshToken(
	token string,
) []byte {
	hash := sha256.Sum256(
		[]byte(token),
	)

	return hash[:]
}
func (s *TokenService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	hash := sha256.Sum256(
		[]byte(refreshToken),
	)

	return s.repo.DeleteRefreshToken(
		ctx,
		hash[:],
	)
}