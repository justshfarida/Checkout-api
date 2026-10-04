package services

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

// fakeAuthRepository is shared by AuthService and TokenService tests.
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

	// FindUserByID
	userByID       models.User
	findByIDErr    error
	findByIDCalled bool
	findByIDArg    int

	// DeleteRefreshToken
	deleteRefreshErr    error
	deleteRefreshCalled bool
	deletedTokenHash    []byte

	// GetUserInventory
	inventory    []models.InventoryItem
	inventoryErr error
	inventoryArg int
}

func (f *fakeAuthRepository) FindUserByID(
	ctx context.Context,
	id int,
) (models.User, error) {
	f.findByIDCalled = true
	f.findByIDArg = id

	if f.findByIDErr != nil {
		return models.User{}, f.findByIDErr
	}

	return f.userByID, nil
}

func (f *fakeAuthRepository) DeleteRefreshToken(
	ctx context.Context,
	tokenHash []byte,
) error {
	f.deleteRefreshCalled = true
	f.deletedTokenHash = append([]byte(nil), tokenHash...)

	return f.deleteRefreshErr
}

func (f *fakeAuthRepository) GetUserInventory(
	ctx context.Context,
	userID int,
) ([]models.InventoryItem, error) {
	f.inventoryArg = userID

	if f.inventoryErr != nil {
		return nil, f.inventoryErr
	}

	return f.inventory, nil
}

var _ repositories.UserRepository = (*fakeAuthRepository)(nil)

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
	f.savedRefreshTokenHash = append([]byte(nil), tokenHash...)
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
	f.oldRefreshTokenHash = append([]byte(nil), oldTokenHash...)
	f.newRefreshTokenHash = append([]byte(nil), newTokenHash...)
	f.newRefreshExpiresAt = newExpiresAt

	if f.rotateRefreshErr != nil {
		return 0, f.rotateRefreshErr
	}

	return f.rotateUserID, nil
}

func TestAuthService_SaveUser(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		password       []byte
		repoErr        error
		wantErr        error
		wantRepoCalled bool
	}{
		{
			// Z — Zero
			name:           "zero - empty password",
			email:          "farida@example.com",
			password:       []byte{},
			wantRepoCalled: true,
		},
		{
			// O — One
			name:           "one - one character password",
			email:          "farida@example.com",
			password:       []byte("a"),
			wantRepoCalled: true,
		},
		{
			// M — Many
			name:           "many - normal multi-character password",
			email:          "farida@example.com",
			password:       []byte("MyStrongPassword123"),
			wantRepoCalled: true,
		},
		{
			// B — bcrypt accepts a 72-byte password.
			name:           "boundary - 72 byte password",
			email:          "farida@example.com",
			password:       bytes.Repeat([]byte("a"), 72),
			wantRepoCalled: true,
		},
		{
			// B + E — bcrypt rejects passwords longer than 72 bytes.
			name:           "boundary - 73 byte password",
			email:          "farida@example.com",
			password:       bytes.Repeat([]byte("a"), 73),
			wantErr:        ErrPasswordHashing,
			wantRepoCalled: false,
		},
		{
			// I — repository error contract.
			name:           "interface - repository returns error",
			email:          "farida@example.com",
			password:       []byte("MyStrongPassword123"),
			repoErr:        errors.New("database unavailable"),
			wantRepoCalled: true,
		},
		{
			// S — simple happy path.
			name:           "simple - valid password saved successfully",
			email:          "farida@example.com",
			password:       []byte("Password123"),
			wantRepoCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAuthRepository{saveUserErr: tt.repoErr}
			service := NewAuthService(repo)

			err := service.SaveUser(
				context.Background(),
				tt.email,
				tt.password,
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else if tt.repoErr != nil {
				if !errors.Is(err, tt.repoErr) {
					t.Fatalf("expected repository error %v, got %v", tt.repoErr, err)
				}
			} else if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if repo.saveUserCalled != tt.wantRepoCalled {
				t.Fatalf(
					"expected repo called = %v, got %v",
					tt.wantRepoCalled,
					repo.saveUserCalled,
				)
			}

			if !tt.wantRepoCalled {
				return
			}

			if repo.savedEmail != tt.email {
				t.Errorf("expected email %q, got %q", tt.email, repo.savedEmail)
			}

			if err := bcrypt.CompareHashAndPassword(
				repo.savedHash,
				tt.password,
			); err != nil {
				t.Error("saved hash does not match original password")
			}
		})
	}
}

func TestAuthService_Authenticate(t *testing.T) {
	password := []byte("Password123")
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to prepare password hash: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		repo := &fakeAuthRepository{
			user: models.User{
				ID:    7,
				Email: "farida@example.com",
				Hash:  hash,
			},
		}
		service := NewAuthService(repo)

		user, err := service.Authenticate(
			context.Background(),
			"farida@example.com",
			password,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if user.ID != 7 {
			t.Errorf("expected user ID 7, got %d", user.ID)
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

	t.Run("user not found becomes invalid credentials", func(t *testing.T) {
		repo := &fakeAuthRepository{
			findUserErr: repositories.ErrUserNotFound,
		}
		service := NewAuthService(repo)

		_, err := service.Authenticate(
			context.Background(),
			"missing@example.com",
			password,
		)

		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		repo := &fakeAuthRepository{
			user: models.User{
				ID:    7,
				Email: "farida@example.com",
				Hash:  hash,
			},
		}
		service := NewAuthService(repo)

		_, err := service.Authenticate(
			context.Background(),
			"farida@example.com",
			[]byte("WrongPassword"),
		)

		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repoErr := errors.New("database unavailable")
		repo := &fakeAuthRepository{findUserErr: repoErr}
		service := NewAuthService(repo)

		_, err := service.Authenticate(
			context.Background(),
			"farida@example.com",
			password,
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repository error %v, got %v", repoErr, err)
		}
	})
}
