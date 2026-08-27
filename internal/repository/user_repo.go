package repository

import (
	"checkout-api/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	r := &UserRepo{
		db: db,
	}
	return r
}

func (r *UserRepo) SignUp(ctx context.Context, username string, passwordHash string) (*models.User, error) {
	var user models.User
	user.Username = username
	user.PasswordHash = passwordHash

	err := r.db.QueryRow(ctx, `insert into users (username, password_hash) values ($1, $2) returning id, created_at`, username, passwordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("unable to create/find cart for user %d: %w")
	}
	if err != nil {
		var pgErr *pgconn.PgError                            // REPLACE WITH PGPOOL ERROR INSTEAD OF CONN
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // username alr exists err
			return nil, fmt.Errorf("username %q already taken", username)
		}
		return nil, fmt.Errorf("unable to create user: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) Login(ctx context.Context, username, password string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx,
		"select id, username, password_hash, created_at from users where username=$1", username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("invalid username or password")
		}
		return nil, fmt.Errorf("unable to fetch user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	return &user, nil
}
