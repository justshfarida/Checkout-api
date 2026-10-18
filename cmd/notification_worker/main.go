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

	amqp "github.com/rabbitmq/amqp091-go"
)

// requeueDelay avoids a hot loop: without it a message whose email keeps
// failing would bounce back and forth hundreds of times per second.
const requeueDelay = 1 * time.Second

func main() {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		log.Fatal("RABBITMQ_URL is not set")
	}

	// WORKER_ID labels this copy in logs; the container hostname is unique
	// per replica, so scaled workers get different ids automatically.
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID, _ = os.Hostname()
	}

	broker, err := messaging.Connect(url)
	if err != nil {
		log.Fatalf("[%s] connect: %v", workerID, err)
	}
	defer broker.Close()

	deliveries, err := broker.Consume(messaging.OrderNotificationsQueue, workerID, 1)
	if err != nil {
		log.Fatalf("[%s] consume: %v", workerID, err)
	}

	// Stop on Ctrl+C / docker stop. The current message is finished first,
	// because we only check for shutdown between messages.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
			process(ctx, workerID, msg)
		}
	}
}

// process applies the acknowledgment decision tree (lecture slide 34):
//
//	success           -> Ack                  (remove from queue)
//	transient failure -> Nack, requeue=true   (try again later)
//	permanent failure -> Nack, requeue=false  (dead-letter queue)
func process(ctx context.Context, workerID string, msg amqp.Delivery) {
	var event messaging.OrderPaidEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil || event.OrderID <= 0 {
		// Broken message: retrying can never fix it.
		log.Printf("[%s] invalid message %q: sending to DLQ", workerID, msg.MessageId)
		_ = msg.Nack(false, false)
		return
	}

	log.Printf("[%s] sending confirmation for order #%d (message %s)", workerID, event.OrderID, msg.MessageId)

	if err := notifier.SendOrderConfirmation(ctx, event); err != nil {
		// Email provider down, timeout, or we are shutting down: try again later.
		log.Printf("[%s] email for order #%d failed: %v - requeueing", workerID, event.OrderID, err)
		time.Sleep(requeueDelay)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
	log.Printf("[%s] done order #%d", workerID, event.OrderID)
}
