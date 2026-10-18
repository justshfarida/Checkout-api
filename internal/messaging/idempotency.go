package messaging

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB is the small part of *pgxpool.Pool the dedup logic needs. An interface
// so tests can use a fake instead of a real database.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// BeginProcessing records that a message is being handled, BEFORE its side
// effect. It returns alreadyDone=true when the same message was already
// processed successfully: a duplicate delivery the caller must skip.
func BeginProcessing(ctx context.Context, db DB, eventType, messageID string) (alreadyDone bool, err error) {
	tag, err := db.Exec(ctx,
		`INSERT INTO processed_messages (event_type, message_id)
		 VALUES ($1, $2)
		 ON CONFLICT (event_type, message_id) DO NOTHING`,
		eventType, messageID,
	)
	if err != nil {
		return false, fmt.Errorf("record message %s: %w", messageID, err)
	}

	// New row: first time we see this message.
	if tag.RowsAffected() == 1 {
		return false, nil
	}

	// Seen before: either finished (duplicate) or an earlier attempt failed
	// or crashed before finishing (must be retried).
	var done bool
	if err := db.QueryRow(ctx,
		`SELECT processed_at IS NOT NULL
		   FROM processed_messages
		  WHERE event_type = $1 AND message_id = $2`,
		eventType, messageID,
	).Scan(&done); err != nil {
		return false, fmt.Errorf("check message %s: %w", messageID, err)
	}
	return done, nil
}

// MarkProcessed records that the side effect succeeded. From now on, any
// redelivery of this message is a duplicate.
func MarkProcessed(ctx context.Context, db DB, eventType, messageID string) error {
	_, err := db.Exec(ctx,
		`UPDATE processed_messages
		    SET processed_at = now()
		  WHERE event_type = $1 AND message_id = $2`,
		eventType, messageID,
	)
	if err != nil {
		return fmt.Errorf("mark message %s processed: %w", messageID, err)
	}
	return nil
}
