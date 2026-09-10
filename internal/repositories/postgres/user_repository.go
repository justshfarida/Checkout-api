package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"checkout-api/internal/models"

	"checkout-api/internal/repositories"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) FindUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	var user models.User

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			email,
			hash
		FROM users
		WHERE email = $1
		`,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Hash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, repositories.ErrUserNotFound
	}

	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (r *PostgresRepository) SaveUser(
	ctx context.Context,
	email string,
	hash []byte,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO users (
			email,
			hash
		)
		VALUES ($1, $2)
		`,
		email,
		hash,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to save user: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresRepository) SaveRefreshToken(
	ctx context.Context,
	userID int,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		`,
		userID,
		tokenHash,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to save refresh token: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresRepository) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash []byte,
	newTokenHash []byte,
	newExpiresAt time.Time,
) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to begin refresh-token transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	var (
		userID       int
		oldExpiresAt time.Time
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			user_id,
			expires_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND is_active = TRUE
		FOR UPDATE
		`,
		oldTokenHash,
	).Scan(
		&userID,
		&oldExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf(
				"invalid or inactive refresh token",
			)
		}

		return 0, fmt.Errorf(
			"failed to retrieve refresh token: %w",
			err,
		)
	}

	if time.Now().After(oldExpiresAt) {
		return 0, fmt.Errorf(
			"refresh token has expired",
		)
	}

	commandTag, err := tx.Exec(
		ctx,
		`
		UPDATE refresh_tokens
		SET is_active = FALSE
		WHERE token_hash = $1
		`,
		oldTokenHash,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to deactivate refresh token: %w",
			err,
		)
	}

	if commandTag.RowsAffected() != 1 {
		return 0, fmt.Errorf(
			"failed to deactivate refresh token",
		)
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		`,
		userID,
		newTokenHash,
		newExpiresAt,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to save new refresh token: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(
			"failed to commit refresh-token rotation: %w",
			err,
		)
	}

	return userID, nil
}
