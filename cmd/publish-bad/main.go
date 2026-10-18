// Command publish-bad sends a poison message (a body the worker cannot
// parse) to order_notifications, to prove it ends up in the DLQ (exercise 3).
//
// Usage: go run ./cmd/publish-bad
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"checkout-api/internal/messaging"
)

func main() {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	broker, err := messaging.Connect(url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer broker.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := fmt.Sprintf("poison:%d", time.Now().Unix())
	body := []byte(`{"order_id":"not-a-number"}`)
	if err := broker.PublishEvent(ctx, "order.paid", id, body); err != nil {
		log.Fatalf("publish: %v", err)
	}
	fmt.Printf("published poison message %s -> %s\n", id, messaging.OrderNotificationsQueue)
}
