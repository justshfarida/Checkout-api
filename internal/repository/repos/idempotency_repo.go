package repos

import (
	"checkout-api/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyRepo struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepo(db *pgxpool.Pool) *IdempotencyRepo {
	r := &IdempotencyRepo{
		db: db,
	}
	return r
}

func (r *IdempotencyRepo) GetIdempotencyRecord(ctx context.Context, key string) (*models.IdempotencyRecord, error) {
	var record models.IdempotencyRecord
	var bodyJSON []byte
	err := r.db.QueryRow(ctx,
		`select request_hash, status_code, body, created_at from idempotency_records
		 where key=$1 and created_at > now() - interval '30 minutes'`, key,
	).Scan(&record.RequestHash, &record.StatusCode, &bodyJSON, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found = not an error
		}
		return nil, fmt.Errorf("unable to fetch idempotency record: %w", err)
	}
	if err := json.Unmarshal(bodyJSON, &record.Body); err != nil {
		return nil, fmt.Errorf("unable to unmarshal idempotency body: %w", err)
	}
	return &record, nil
}

func (r *IdempotencyRepo) SaveIdempotencyRecord(ctx context.Context, key, requestHash string, statusCode int, body any) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("unable to marshal idempotency body: %w", err)
	}
	_, err = r.db.Exec(ctx,
		`insert into idempotency_records (key, request_hash, status_code, body) values ($1, $2, $3, $4)
		 on conflict (key) do update set request_hash=excluded.request_hash,
		   status_code=excluded.status_code, body=excluded.body, created_at=now()`,
		key, requestHash, statusCode, bodyJSON)
	if err != nil {
		return fmt.Errorf("unable to save idempotency record: %w", err)
	}
	return nil
}
