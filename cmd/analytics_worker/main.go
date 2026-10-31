// analytics-worker consumes its own copy of every order.paid event (from the
// order.events fanout exchange) and keeps paid orders and revenue per day in
// the daily_sales table (exercise 4). It is independent of the email worker:
// if it is down, its messages wait in order_analytics and emails still go out.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"checkout-api/internal/analytics"
	"checkout-api/internal/messaging"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	retryDelay         = 1 * time.Second
	maxAttempts        = 3
	eventTypeOrderPaid = "order.paid"
)

// recordFunc stores one paid order; returns duplicate=true if already counted.
type recordFunc func(ctx context.Context, messageID string, e messaging.OrderPaidEvent) (analytics.DayTotals, bool, error)

// worker holds what process needs, so tests can swap in fakes.
type worker struct {
	id         string
	record     recordFunc
	retry      func(ctx context.Context, msg amqp.Delivery, attempt int) error
	retryDelay time.Duration
}

func main() {
	rabbitURL := mustEnv("RABBITMQ_URL")
	databaseURL := mustEnv("DATABASE_URL")

	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID, _ = os.Hostname()
	}
	workerID = "analytics-" + workerID

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("[%s] database: %v", workerID, err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		log.Fatalf("[%s] database ping: %v", workerID, err)
	}

	broker, err := messaging.Connect(rabbitURL)
	if err != nil {
		log.Fatalf("[%s] connect: %v", workerID, err)
	}
	defer broker.Close()

	deliveries, err := broker.Consume(messaging.OrderAnalyticsQueue, workerID, 1)
	if err != nil {
		log.Fatalf("[%s] consume: %v", workerID, err)
	}

	w := &worker{
		id: workerID,
		record: func(ctx context.Context, messageID string, e messaging.OrderPaidEvent) (analytics.DayTotals, bool, error) {
			return analytics.RecordPaidOrder(ctx, pool, eventTypeOrderPaid, messageID, e.Total, e.PaidAt)
		},
		// Retries go straight to OUR queue, not to the fanout exchange:
		// the email worker already handled its own copy.
		retry: func(ctx context.Context, msg amqp.Delivery, attempt int) error {
			return broker.Republish(ctx, messaging.OrderAnalyticsQueue, msg,
				amqp.Table{messaging.RetryCountHeader: int32(attempt)})
		},
		retryDelay: retryDelay,
	}

	log.Printf("[%s] waiting for messages on %s", workerID, messaging.OrderAnalyticsQueue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] shutting down", workerID)
			return
		case msg, ok := <-deliveries:
			if !ok {
				log.Fatalf("[%s] delivery channel closed", workerID)
			}
			w.process(ctx, msg)
		}
	}
}

// process: same decision tree as the email worker, but dedup and the work
// happen in one database transaction (see analytics.RecordPaidOrder).
//
//	broken message          -> Nack, requeue=false  (dead-letter queue)
//	already counted         -> Ack                  (duplicate: skip)
//	counted                 -> Ack
//	failure, tries left     -> retry later with attempt+1
//	failure, no tries left  -> Nack, requeue=false  (dead-letter queue)
func (w *worker) process(ctx context.Context, msg amqp.Delivery) {
	var event messaging.OrderPaidEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil || event.OrderID <= 0 || msg.MessageId == "" {
		log.Printf("[%s] invalid message %q: sending to DLQ", w.id, msg.MessageId)
		_ = msg.Nack(false, false)
		return
	}

	totals, duplicate, err := w.record(ctx, msg.MessageId, event)
	if err != nil {
		w.retryOrDeadLetter(ctx, msg, err.Error())
		return
	}
	if duplicate {
		log.Printf("[%s] duplicate %s: already counted, skipping", w.id, msg.MessageId)
		_ = msg.Ack(false)
		return
	}

	_ = msg.Ack(false)
	log.Printf("[%s] counted order #%d (total %d): %s now has %d orders, revenue %d",
		w.id, event.OrderID, event.Total, totals.Day, totals.Orders, totals.Revenue)
}

// retryOrDeadLetter: retry with a counter, then give up into the DLQ.
func (w *worker) retryOrDeadLetter(ctx context.Context, msg amqp.Delivery, reason string) {
	if ctx.Err() != nil {
		_ = msg.Nack(false, true)
		return
	}

	attempt := retryCount(msg) + 1
	if attempt >= maxAttempts {
		log.Printf("[%s] %s: try %d/%d, giving up -> DLQ (%s)", w.id, msg.MessageId, attempt, maxAttempts, reason)
		_ = msg.Nack(false, false)
		return
	}

	log.Printf("[%s] %s: try %d/%d failed (%s), retrying", w.id, msg.MessageId, attempt, maxAttempts, reason)
	time.Sleep(w.retryDelay)

	if err := w.retry(ctx, msg, attempt); err != nil {
		log.Printf("[%s] retry publish for %s failed: %v - requeueing", w.id, msg.MessageId, err)
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

func retryCount(msg amqp.Delivery) int {
	switch v := msg.Headers[messaging.RetryCountHeader].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is not set", key)
	}
	return value
}
