// notification-worker consumes order.paid events and sends confirmation
// emails. Run several copies to share the load: RabbitMQ gives each message
// to exactly one of them (competing consumers).
package main

import (
	"context"
	"encoding/json"
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
	// requeueDelay avoids a hot loop: without it a message whose email keeps
	// failing would bounce back and forth hundreds of times per second.
	requeueDelay = 1 * time.Second

	// eventTypeOrderPaid is part of the dedup key in processed_messages.
	eventTypeOrderPaid = "order.paid"
)

// worker holds what process needs, so tests can swap in fakes.
type worker struct {
	id           string
	db           messaging.DB
	sendEmail    func(context.Context, messaging.OrderPaidEvent) error
	requeueDelay time.Duration
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

	w := &worker{
		id:           workerID,
		db:           pool,
		sendEmail:    notifier.SendOrderConfirmation,
		requeueDelay: requeueDelay,
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
// with an idempotency check before the side effect (slide 39):
//
//	broken message      -> Nack, requeue=false  (dead-letter queue)
//	already processed   -> Ack                  (duplicate: skip)
//	email sent          -> mark processed, Ack
//	transient failure   -> Nack, requeue=true   (try again later)
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
		log.Printf("[%s] dedup check for %s failed: %v - requeueing", w.id, msg.MessageId, err)
		w.requeue(msg)
		return
	}
	if alreadyDone {
		log.Printf("[%s] duplicate %s: already processed, skipping", w.id, msg.MessageId)
		_ = msg.Ack(false)
		return
	}

	log.Printf("[%s] sending confirmation for order #%d (message %s)", w.id, event.OrderID, msg.MessageId)

	if err := w.sendEmail(ctx, event); err != nil {
		// Email provider down, timeout, or shutting down. The row stays
		// "unfinished", so the retry will send the email.
		log.Printf("[%s] email for order #%d failed: %v - requeueing", w.id, event.OrderID, err)
		w.requeue(msg)
		return
	}

	if err := messaging.MarkProcessed(ctx, w.db, eventTypeOrderPaid, msg.MessageId); err != nil {
		// The email already went out. Ack anyway: requeueing would send it
		// again right away. Logged so it can be investigated.
		log.Printf("[%s] email sent but marking %s failed: %v", w.id, msg.MessageId, err)
	}

	_ = msg.Ack(false)
	log.Printf("[%s] done order #%d", w.id, event.OrderID)
}

// requeue waits a moment, then puts the message back in the queue.
func (w *worker) requeue(msg amqp.Delivery) {
	time.Sleep(w.requeueDelay)
	_ = msg.Nack(false, true)
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is not set", key)
	}
	return value
}
