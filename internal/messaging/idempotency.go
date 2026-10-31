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
// processed successfully by this consumer: a duplicate delivery to skip.
//
// consumer (e.g. "email", "analytics") is part of the key: with a fanout
// exchange every consumer gets its own copy of the same message id, and one
// consumer finishing must not make the others skip theirs (exercise 4).
func BeginProcessing(ctx context.Context, db DB, consumer, eventType, messageID string) (alreadyDone bool, err error) {
	tag, err := db.Exec(ctx,
		`INSERT INTO processed_messages (consumer, event_type, message_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (consumer, event_type, message_id) DO NOTHING`,
		consumer, eventType, messageID,
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
		  WHERE consumer = $1 AND event_type = $2 AND message_id = $3`,
		consumer, eventType, messageID,
	).Scan(&done); err != nil {
		return false, fmt.Errorf("check message %s: %w", messageID, err)
	}
	return done, nil
}

// MarkProcessed records that the side effect succeeded. From now on, any
// redelivery of this message is a duplicate.
func MarkProcessed(ctx context.Context, db DB, consumer, eventType, messageID string) error {
	_, err := db.Exec(ctx,
		`UPDATE processed_messages
		    SET processed_at = now()
		  WHERE consumer = $1 AND event_type = $2 AND message_id = $3`,
		consumer, eventType, messageID,
	)
	if err != nil {
		return fmt.Errorf("mark message %s processed: %w", messageID, err)
	}
	return nil
}
