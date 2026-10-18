// Package analytics turns order.paid events into daily sales numbers.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Consumer is the analytics worker's name in processed_messages.
const Consumer = "analytics"

// DayTotals is one row of daily_sales after an update.
type DayTotals struct {
	Day     string
	Orders  int
	Revenue int64
}

// TxStarter is the part of *pgxpool.Pool this package needs.
type TxStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// RecordPaidOrder adds one paid order to daily_sales.
//
// The dedup row and the counter update are written in ONE transaction:
// either both happen or neither does. So a redelivered message can never
// count an order twice, and a crash in the middle leaves no half-done
// state: "exactly-once effect" on top of at-least-once delivery.
//
// This is stronger than the email worker, whose side effect (an email)
// cannot be part of a database transaction.
func RecordPaidOrder(ctx context.Context, db TxStarter, eventType, messageID string, total int, paidAt time.Time) (totals DayTotals, duplicate bool, err error) {
	if paidAt.IsZero() {
		paidAt = time.Now()
	}
	day := paidAt.UTC().Format("2006-01-02")

	tx, err := db.Begin(ctx)
	if err != nil {
		return DayTotals{}, false, fmt.Errorf("begin: %w", err)
	}
	// Rollback after Commit is a no-op, so this is safe on every path.
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`INSERT INTO processed_messages (consumer, event_type, message_id, processed_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (consumer, event_type, message_id) DO NOTHING`,
		Consumer, eventType, messageID,
	)
	if err != nil {
		return DayTotals{}, false, fmt.Errorf("record message %s: %w", messageID, err)
	}
	if tag.RowsAffected() == 0 {
		// Already counted in an earlier, committed transaction.
		return DayTotals{}, true, nil
	}

	totals.Day = day
	if err := tx.QueryRow(ctx,
		`INSERT INTO daily_sales (day, orders, revenue)
		 VALUES ($1::date, 1, $2)
		 ON CONFLICT (day) DO UPDATE
		    SET orders     = daily_sales.orders + 1,
		        revenue    = daily_sales.revenue + EXCLUDED.revenue,
		        updated_at = now()
		 RETURNING orders, revenue`,
		day, total,
	).Scan(&totals.Orders, &totals.Revenue); err != nil {
		return DayTotals{}, false, fmt.Errorf("update daily_sales: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DayTotals{}, false, fmt.Errorf("commit: %w", err)
	}
	return totals, false, nil
}
