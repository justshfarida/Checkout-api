package services

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

func TestAuthService_UserExists(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		want    bool
		wantErr error
	}{
		{name: "existing user returns true", want: true},
		{name: "not found is reported as false without error", repoErr: repositories.ErrUserNotFound, want: false},
		{name: "infrastructure error is returned", repoErr: errFakeDependency, want: false, wantErr: errFakeDependency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAuthRepository{userByID: models.User{ID: 5}, findByIDErr: tt.repoErr}
			service := NewAuthService(repo)

			got, err := service.UserExists(context.Background(), 5)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UserExists() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("UserExists() = %v, want %v", got, tt.want)
			}
			if repo.findByIDArg != 5 {
				t.Errorf("looked up user %d, want 5", repo.findByIDArg)
			}
		})
	}
}

func TestTokenService_Logout_DeletesTheStoredRefreshToken(t *testing.T) {
	repo := &fakeAuthRepository{}
	service := NewTokenService(repo, []byte("secret"))

	token, _, err := service.CreateRefreshToken(context.Background(), 3)
	if err != nil {
		t.Fatalf("CreateRefreshToken() error = %v", err)
	}

	if err := service.Logout(context.Background(), token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	// Logout must target the exact row that login stored, otherwise the
	// refresh token would stay valid after logout.
	if !bytes.Equal(repo.deletedTokenHash, repo.savedRefreshTokenHash) {
		t.Error("Logout deleted a different hash than the one saved for the refresh token")
	}
}

func TestTokenService_Logout_ReturnsRepositoryError(t *testing.T) {
	repo := &fakeAuthRepository{deleteRefreshErr: errFakeDependency}
	service := NewTokenService(repo, []byte("secret"))

	if err := service.Logout(context.Background(), "tok"); !errors.Is(err, errFakeDependency) {
		t.Fatalf("Logout() error = %v, want %v", err, errFakeDependency)
	}
}
