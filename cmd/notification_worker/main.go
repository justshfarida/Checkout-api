// notification-worker consumes order.paid events and sends confirmation
// emails. Run several copies to share the load: RabbitMQ gives each message
// to exactly one of them (competing consumers).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"checkout-api/internal/messaging"
	"checkout-api/internal/notifier"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// retryDelay is the pause before a failed message is tried again, so a
	// failing email provider is not hammered hundreds of times per second.
	retryDelay = 1 * time.Second

	// maxAttempts: after this many failed tries the message is a poison
	// message and goes to the dead-letter queue (exercise 3).
	maxAttempts = 3

	// eventTypeOrderPaid is part of the dedup key in processed_messages.
	eventTypeOrderPaid = "order.paid"
)

// worker holds what process needs, so tests can swap in fakes.
type worker struct {
	id         string
	db         messaging.DB
	sendEmail  func(context.Context, messaging.OrderPaidEvent) error
	retry      func(ctx context.Context, msg amqp.Delivery, attempt int) error
	retryDelay time.Duration
}

func main() {
	rabbitURL := mustEnv("RABBITMQ_URL")
	databaseURL := mustEnv("DATABASE_URL")

	// WORKER_ID labels this copy in logs; the container hostname is unique
	// per replica, so scaled workers get different ids automatically.
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID, _ = os.Hostname()
	}

	// Stop on Ctrl+C / docker stop. The current message is finished first,
	// because we only check for shutdown between messages.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database: remembers which messages were already handled (exercise 1).
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

	deliveries, err := broker.Consume(messaging.OrderNotificationsQueue, workerID, 1)
	if err != nil {
		log.Fatalf("[%s] consume: %v", workerID, err)
	}

	sendEmail := notifier.SendOrderConfirmation
	// Demo switch for exercise 3: pretend the email provider is down, so
	// messages exhaust their retries and land in the dead-letter queue.
	if os.Getenv("SIMULATE_EMAIL_FAILURE") == "true" {
		log.Printf("[%s] SIMULATE_EMAIL_FAILURE=true: every email will fail", workerID)
		sendEmail = func(context.Context, messaging.OrderPaidEvent) error {
			return errors.New("simulated email provider outage")
		}
	}

	w := &worker{
		id:        workerID,
		db:        pool,
		sendEmail: sendEmail,
		// A retry is a new copy of the message with the attempt counter in a
		// header; RabbitMQ itself does not count requeues on classic queues.
		retry: func(ctx context.Context, msg amqp.Delivery, attempt int) error {
			return broker.Republish(ctx, messaging.OrderNotificationsQueue, msg,
				amqp.Table{messaging.RetryCountHeader: int32(attempt)})
		},
		retryDelay: retryDelay,
	}

	log.Printf("[%s] waiting for messages on %s", workerID, messaging.OrderNotificationsQueue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] shutting down", workerID)
			return

		case msg, ok := <-deliveries:
			if !ok {
				// The broker closed the connection. Exit so Docker restarts us
				// (restart: unless-stopped) and we reconnect cleanly.
				log.Fatalf("[%s] delivery channel closed", workerID)
			}
			w.process(ctx, msg)
		}
	}
}

// process applies the acknowledgment decision tree (lecture slide 34),
// with an idempotency check before the side effect (slide 39) and a retry
// limit before the dead-letter queue (slide 20):
//
//	broken message          -> Nack, requeue=false  (dead-letter queue)
//	already processed       -> Ack                  (duplicate: skip)
//	email sent              -> mark processed, Ack
//	failure, tries left     -> retry later with attempt+1
//	failure, no tries left  -> Nack, requeue=false  (dead-letter queue)
func (w *worker) process(ctx context.Context, msg amqp.Delivery) {
	var event messaging.OrderPaidEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil || event.OrderID <= 0 || msg.MessageId == "" {
		// Broken message: retrying can never fix it. Without a MessageId we
		// also could not recognise duplicates.
		log.Printf("[%s] invalid message %q: sending to DLQ", w.id, msg.MessageId)
		_ = msg.Nack(false, false)
		return
	}

	// Exercise 1: record the message BEFORE the side effect.
	alreadyDone, err := messaging.BeginProcessing(ctx, w.db, eventTypeOrderPaid, msg.MessageId)
	if err != nil {
		w.retryOrDeadLetter(ctx, msg, "dedup check failed: "+err.Error())
		return
	}
	if alreadyDone {
		log.Printf("[%s] duplicate %s: already processed, skipping", w.id, msg.MessageId)
		_ = msg.Ack(false)
		return
	}

	log.Printf("[%s] sending confirmation for order #%d to user %d (message %s)",
		w.id, event.OrderID, event.UserID, msg.MessageId)

	if err := w.sendEmail(ctx, event); err != nil {
		// The processed_messages row stays "unfinished", so a retry sends it.
		w.retryOrDeadLetter(ctx, msg, "email failed: "+err.Error())
		return
	}

	if err := messaging.MarkProcessed(ctx, w.db, eventTypeOrderPaid, msg.MessageId); err != nil {
		// The email already went out. Ack anyway: retrying would send it
		// again right away. Logged so it can be investigated.
		log.Printf("[%s] email sent but marking %s failed: %v", w.id, msg.MessageId, err)
	}

	_ = msg.Ack(false)
	log.Printf("[%s] done order #%d", w.id, event.OrderID)
}

// retryOrDeadLetter handles a failed attempt. While tries are left, a copy
// with attempt+1 goes back to the queue and the original is acked. After
// maxAttempts the message is rejected and RabbitMQ routes it to the DLQ.
func (w *worker) retryOrDeadLetter(ctx context.Context, msg amqp.Delivery, reason string) {
	// Shutting down: put it back untouched, this attempt does not count.
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
		// Could not publish the retry copy: fall back to a plain requeue so
		// the message is never lost.
		log.Printf("[%s] retry publish for %s failed: %v - requeueing", w.id, msg.MessageId, err)
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

// retryCount reads how many times this message was already retried.
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
