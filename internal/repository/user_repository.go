package repository

import (
	"checkout-api/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) SignUp(ctx context.Context, username string, passwordHash string) (*models.User, error) {
	var user models.User
	user.Username = username
	user.PasswordHash = passwordHash

	err := r.db.QueryRow(ctx,
		`insert into users (username, password_hash) values ($1, $2) returning id, created_at`,
		username, passwordHash,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("unable to create user: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) Login(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx,
		"select id, username, password_hash, created_at from users where username=$1", username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("unable to fetch user %q: %w", username, err)
	}
	return &user, nil
}
