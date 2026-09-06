package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"checkout-api/internal/models"
)

// Fake repository used only by AuthService tests.
type fakeAuthRepository struct {
	user models.User

	findUserErr         error
	saveUserErr         error
	saveRefreshTokenErr error
	rotateRefreshErr    error

	rotateUserID int

	findUserCalled         bool
	saveUserCalled         bool
	saveRefreshTokenCalled bool
	rotateRefreshCalled    bool

	savedEmail string
	savedHash  []byte

	savedRefreshUserID    int
	savedRefreshTokenHash []byte
	savedRefreshExpiresAt time.Time

	oldRefreshTokenHash []byte
	newRefreshTokenHash []byte
	newRefreshExpiresAt time.Time
}

func (f *fakeAuthRepository) FindUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	f.findUserCalled = true
	f.savedEmail = email

	if f.findUserErr != nil {
		return models.User{}, f.findUserErr
	}

	return f.user, nil
}

func (f *fakeAuthRepository) SaveUser(
	ctx context.Context,
	email string,
	hash []byte,
) error {
	f.saveUserCalled = true
	f.savedEmail = email
	f.savedHash = hash

	return f.saveUserErr
}

func (f *fakeAuthRepository) SaveRefreshToken(
	ctx context.Context,
	userID int,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	f.saveRefreshTokenCalled = true
	f.savedRefreshUserID = userID
	f.savedRefreshTokenHash = tokenHash
	f.savedRefreshExpiresAt = expiresAt

	return f.saveRefreshTokenErr
}

func (f *fakeAuthRepository) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash []byte,
	newTokenHash []byte,
	newExpiresAt time.Time,
) (int, error) {
	f.rotateRefreshCalled = true
	f.oldRefreshTokenHash = oldTokenHash
	f.newRefreshTokenHash = newTokenHash
	f.newRefreshExpiresAt = newExpiresAt

	if f.rotateRefreshErr != nil {
		return 0, f.rotateRefreshErr
	}

	return f.rotateUserID, nil
}

func TestAuthService_FindUserByEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{
			user: models.User{
				ID:    1,
				Email: "farida@example.com",
				Hash:  []byte("hashed-password"),
			},
		}

		service := NewAuthService(repo)

		user, err := service.FindUserByEmail(
			context.Background(),
			"farida@example.com",
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if user.ID != 1 {
			t.Errorf("expected user ID 1, got %d", user.ID)
		}

		if user.Email != "farida@example.com" {
			t.Errorf(
				"expected email farida@example.com, got %s",
				user.Email,
			)
		}

		if !repo.findUserCalled {
			t.Fatal("expected FindUserByEmail repository method to be called")
		}

		if repo.savedEmail != "farida@example.com" {
			t.Errorf(
				"expected repository to receive farida@example.com, got %s",
				repo.savedEmail,
			)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repoErr := errors.New("database unavailable")

		repo := &fakeAuthRepository{
			findUserErr: repoErr,
		}

		service := NewAuthService(repo)

		_, err := service.FindUserByEmail(
			context.Background(),
			"farida@example.com",
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"expected repository error %v, got %v",
				repoErr,
				err,
			)
		}

		if !repo.findUserCalled {
			t.Fatal("expected repository to be called")
		}
	})
}

func TestAuthService_SaveUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{}
		service := NewAuthService(repo)

		hash := []byte("hashed-password")

		err := service.SaveUser(
			context.Background(),
			"farida@example.com",
			hash,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if !repo.saveUserCalled {
			t.Fatal("expected SaveUser repository method to be called")
		}

		if repo.savedEmail != "farida@example.com" {
			t.Errorf(
				"expected email farida@example.com, got %s",
				repo.savedEmail,
			)
		}

		if string(repo.savedHash) != string(hash) {
			t.Errorf(
				"expected hash %q, got %q",
				string(hash),
				string(repo.savedHash),
			)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repoErr := errors.New("failed to save user")

		repo := &fakeAuthRepository{
			saveUserErr: repoErr,
		}

		service := NewAuthService(repo)

		err := service.SaveUser(
			context.Background(),
			"farida@example.com",
			[]byte("hash"),
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"expected %v, got %v",
				repoErr,
				err,
			)
		}

		if !repo.saveUserCalled {
			t.Fatal("expected SaveUser repository method to be called")
		}
	})
}

func TestAuthService_SaveRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{}
		service := NewAuthService(repo)

		tokenHash := []byte("refresh-token-hash")
		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		err := service.SaveRefreshToken(
			context.Background(),
			7,
			tokenHash,
			expiresAt,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if !repo.saveRefreshTokenCalled {
			t.Fatal(
				"expected SaveRefreshToken repository method to be called",
			)
		}

		if repo.savedRefreshUserID != 7 {
			t.Errorf(
				"expected user ID 7, got %d",
				repo.savedRefreshUserID,
			)
		}

		if string(repo.savedRefreshTokenHash) != string(tokenHash) {
			t.Errorf(
				"expected token hash %q, got %q",
				tokenHash,
				repo.savedRefreshTokenHash,
			)
		}

		if !repo.savedRefreshExpiresAt.Equal(expiresAt) {
			t.Errorf(
				"expected expiration %v, got %v",
				expiresAt,
				repo.savedRefreshExpiresAt,
			)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repoErr := errors.New("failed to save refresh token")

		repo := &fakeAuthRepository{
			saveRefreshTokenErr: repoErr,
		}

		service := NewAuthService(repo)

		err := service.SaveRefreshToken(
			context.Background(),
			7,
			[]byte("hash"),
			time.Now(),
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"expected %v, got %v",
				repoErr,
				err,
			)
		}

		if !repo.saveRefreshTokenCalled {
			t.Fatal(
				"expected SaveRefreshToken repository method to be called",
			)
		}
	})
}

func TestAuthService_RotateRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{
			rotateUserID: 7,
		}

		service := NewAuthService(repo)

		oldHash := []byte("old-token-hash")
		newHash := []byte("new-token-hash")
		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		userID, err := service.RotateRefreshToken(
			context.Background(),
			oldHash,
			newHash,
			expiresAt,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if userID != 7 {
			t.Errorf(
				"expected user ID 7, got %d",
				userID,
			)
		}

		if !repo.rotateRefreshCalled {
			t.Fatal(
				"expected RotateRefreshToken repository method to be called",
			)
		}

		if string(repo.oldRefreshTokenHash) != string(oldHash) {
			t.Errorf(
				"expected old hash %q, got %q",
				oldHash,
				repo.oldRefreshTokenHash,
			)
		}

		if string(repo.newRefreshTokenHash) != string(newHash) {
			t.Errorf(
				"expected new hash %q, got %q",
				newHash,
				repo.newRefreshTokenHash,
			)
		}

		if !repo.newRefreshExpiresAt.Equal(expiresAt) {
			t.Errorf(
				"expected expiration %v, got %v",
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

		service := NewAuthService(repo)

		_, err := service.RotateRefreshToken(
			context.Background(),
			[]byte("old"),
			[]byte("new"),
			time.Now(),
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"expected %v, got %v",
				repoErr,
				err,
			)
		}

		if !repo.rotateRefreshCalled {
			t.Fatal(
				"expected RotateRefreshToken repository method to be called",
			)
		}
	})
}
