package services

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenService_CreateAccessToken(t *testing.T) {
	secret := []byte("test-secret")
	repo := &fakeAuthRepository{}
	service := NewTokenService(repo, secret)

	tokenString, err := service.CreateAccessToken(7)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		},
	)
	if err != nil {
		t.Fatalf("failed to parse generated JWT: %v", err)
	}

	if !token.Valid {
		t.Fatal("expected generated JWT to be valid")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Fatal("expected RegisteredClaims")
	}

	if claims.Subject != strconv.Itoa(7) {
		t.Errorf("expected subject 7, got %q", claims.Subject)
	}

	if claims.IssuedAt == nil {
		t.Fatal("expected IssuedAt claim")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt claim")
	}
	if !claims.ExpiresAt.Time.After(claims.IssuedAt.Time) {
		t.Fatal("expected ExpiresAt to be after IssuedAt")
	}
}

func TestTokenService_CreateRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{}
		service := NewTokenService(repo, []byte("test-secret"))

		before := time.Now()
		refreshToken, expiresAt, err := service.CreateRefreshToken(
			context.Background(),
			7,
		)
		after := time.Now()

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if refreshToken == "" {
			t.Fatal("expected non-empty refresh token")
		}
		if !repo.saveRefreshTokenCalled {
			t.Fatal("expected SaveRefreshToken repository method to be called")
		}
		if repo.savedRefreshUserID != 7 {
			t.Errorf("expected user ID 7, got %d", repo.savedRefreshUserID)
		}

		expectedHash := hashRefreshToken(refreshToken)
		if !bytes.Equal(repo.savedRefreshTokenHash, expectedHash) {
			t.Fatal("repository received an incorrect refresh-token hash")
		}
		if bytes.Equal(repo.savedRefreshTokenHash, []byte(refreshToken)) {
			t.Fatal("repository should store the hash, not the raw token")
		}
		if !repo.savedRefreshExpiresAt.Equal(expiresAt) {
			t.Errorf(
				"expected repository expiry %v, got %v",
				expiresAt,
				repo.savedRefreshExpiresAt,
			)
		}

		minExpiry := before.Add(refreshTokenDuration)
		maxExpiry := after.Add(refreshTokenDuration)
		if expiresAt.Before(minExpiry) || expiresAt.After(maxExpiry) {
			t.Errorf(
				"expiry %v is outside expected range [%v, %v]",
				expiresAt,
				minExpiry,
				maxExpiry,
			)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repoErr := errors.New("failed to save refresh token")
		repo := &fakeAuthRepository{
			saveRefreshTokenErr: repoErr,
		}
		service := NewTokenService(repo, []byte("test-secret"))

		token, expiresAt, err := service.CreateRefreshToken(
			context.Background(),
			7,
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repository error %v, got %v", repoErr, err)
		}
		if token != "" {
			t.Errorf("expected empty token on error, got %q", token)
		}
		if !expiresAt.IsZero() {
			t.Errorf("expected zero expiry on error, got %v", expiresAt)
		}
		if !repo.saveRefreshTokenCalled {
			t.Fatal("expected SaveRefreshToken repository method to be called")
		}
	})
}

func TestTokenService_RotateRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{
			rotateUserID: 7,
		}
		service := NewTokenService(repo, []byte("test-secret"))

		oldRefreshToken := "old-refresh-token"

		userID, newRefreshToken, expiresAt, err :=
			service.RotateRefreshToken(
				context.Background(),
				oldRefreshToken,
			)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if userID != 7 {
			t.Errorf("expected user ID 7, got %d", userID)
		}
		if newRefreshToken == "" {
			t.Fatal("expected a new refresh token")
		}
		if newRefreshToken == oldRefreshToken {
			t.Fatal("expected refresh token to be rotated")
		}
		if !repo.rotateRefreshCalled {
			t.Fatal("expected RotateRefreshToken repository method to be called")
		}

		expectedOldHash := hashRefreshToken(oldRefreshToken)
		if !bytes.Equal(repo.oldRefreshTokenHash, expectedOldHash) {
			t.Fatal("repository received incorrect old token hash")
		}

		expectedNewHash := hashRefreshToken(newRefreshToken)
		if !bytes.Equal(repo.newRefreshTokenHash, expectedNewHash) {
			t.Fatal("repository received incorrect new token hash")
		}

		if !repo.newRefreshExpiresAt.Equal(expiresAt) {
			t.Errorf(
				"expected expiry %v, got %v",
				expiresAt,
				repo.newRefreshExpiresAt,
			)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repoErr := errors.New("refresh token rotation failed")
		repo := &fakeAuthRepository{
			rotateRefreshErr: repoErr,
		}
		service := NewTokenService(repo, []byte("test-secret"))

		userID, newToken, expiresAt, err :=
			service.RotateRefreshToken(
				context.Background(),
				"old-refresh-token",
			)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repository error %v, got %v", repoErr, err)
		}
		if userID != 0 {
			t.Errorf("expected user ID 0, got %d", userID)
		}
		if newToken != "" {
			t.Errorf("expected empty token on error, got %q", newToken)
		}
		if !expiresAt.IsZero() {
			t.Errorf("expected zero expiry on error, got %v", expiresAt)
		}
		if !repo.rotateRefreshCalled {
			t.Fatal("expected RotateRefreshToken repository method to be called")
		}
	})
}
