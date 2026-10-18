// Package outbox implements the transactional outbox relay (exercise 2):
// events are written to the outbox table inside the business transaction,
// and a background relay publishes them to RabbitMQ afterwards.
package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Message is one row of the outbox table.
type Message struct {
	ID        int64
	EventType string
	MessageID string
	Payload   []byte
	CreatedAt time.Time
}

// Store gives the relay access to unsent outbox rows.
type Store interface {
	// ProcessBatch locks up to limit unsent rows, calls publish for each in
	// order, and records the results. It stops at the first failure so the
	// remaining rows keep their order and are retried on the next tick.
	ProcessBatch(ctx context.Context, limit int, publish func(Message) error) (sent int, failed int, err error)
}

// PostgresStore is the real Store, backed by the outbox table.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ProcessBatch(
	ctx context.Context,
	limit int,
	publish func(Message) error,
) (int, int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin outbox batch: %w", err)
	}
	defer tx.Rollback(ctx)

	// FOR UPDATE SKIP LOCKED: if several API replicas run a relay, each one
	// takes different rows instead of publishing the same event twice.
	rows, err := tx.Query(ctx, `
		SELECT id, event_type, message_id, payload, created_at
		  FROM outbox
		 WHERE sent_at IS NULL
		 ORDER BY id
		 LIMIT $1
		   FOR UPDATE SKIP LOCKED`,
		limit,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("select unsent outbox rows: %w", err)
	}

	var batch []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.EventType, &m.MessageID, &m.Payload, &m.CreatedAt); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan outbox row: %w", err)
		}
		batch = append(batch, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("read outbox rows: %w", err)
	}

	sent, failed := 0, 0
	for _, m := range batch {
		if pubErr := publish(m); pubErr != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE outbox
				   SET attempts = attempts + 1, last_error = $2
				 WHERE id = $1`,
				m.ID, pubErr.Error(),
			); err != nil {
				return sent, failed, fmt.Errorf("record outbox failure: %w", err)
			}
			failed++
			// The broker is probably down: stop here and keep the order.
			break
		}

		if _, err := tx.Exec(ctx, `
			UPDATE outbox
			   SET sent_at = now(), attempts = attempts + 1, last_error = NULL
			 WHERE id = $1`,
			m.ID,
		); err != nil {
			return sent, failed, fmt.Errorf("mark outbox row sent: %w", err)
		}
		sent++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit outbox batch: %w", err)
	}
	return sent, failed, nil
}
