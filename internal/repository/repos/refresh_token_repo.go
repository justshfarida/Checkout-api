package repos

import (
	"checkout-api/internal/models"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepo struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	r := &RefreshTokenRepo{
		db: db,
	}
	return r
}

func (r *RefreshTokenRepo) SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`insert into refresh_tokens (token, user_id, expires_at) values ($1, $2, $3)`,
		token, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("unable to save refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{Token: token}
	err := r.db.QueryRow(ctx,
		`select user_id, expires_at, revoked, created_at from refresh_tokens where token=$1`, token,
	).Scan(&rt.UserID, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, fmt.Errorf("unable to fetch refresh token: %w", err)
	}
	return rt, nil
}

func (r *RefreshTokenRepo) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `update refresh_tokens set revoked=true where token=$1`, token)
	if err != nil {
		return fmt.Errorf("unable to revoke refresh token: %w", err)
	}
	return nil
}
