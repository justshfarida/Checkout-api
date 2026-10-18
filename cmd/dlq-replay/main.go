// Command dlq-replay inspects the dead-letter queue and can move messages
// back to the main queue once the problem is fixed (exercise 3).
//
// Usage:
//
//	go run ./cmd/dlq-replay              # list DLQ messages, change nothing
//	go run ./cmd/dlq-replay -replay 5    # move up to 5 messages back
//	go run ./cmd/dlq-replay -replay -1   # move all of them back
//	go run ./cmd/dlq-replay -queue order_analytics   # the analytics DLQ
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"checkout-api/internal/messaging"
)

func main() {
	replay := flag.Int("replay", 0, "number of messages to move back to the main queue (-1 = all, 0 = only list)")
	queue := flag.String("queue", messaging.OrderNotificationsQueue, "main queue; its DLQ is <queue>.dlq")
	flag.Parse()
	dlq := *queue + ".dlq"

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	broker, err := messaging.Connect(url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer broker.Close()

	depth, err := broker.QueueDepth(dlq)
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Printf("%s has %d message(s)\n", dlq, depth)

	if *replay == 0 {
		list(broker, dlq, depth)
		return
	}
	limit := *replay
	if limit < 0 || limit > depth {
		limit = depth
	}
	move(broker, *queue, dlq, limit)
}

// list prints every DLQ message and puts them all back untouched.
// Messages stay unacked while we look, so Get never returns the same one twice.
func list(broker *messaging.RabbitMQ, dlq string, depth int) {
	var held []amqp.Delivery
	defer func() {
		for _, d := range held {
			_ = d.Nack(false, true) // back to the DLQ, unchanged
		}
	}()

	for i := 0; i < depth; i++ {
		d, ok, err := broker.Get(dlq)
		if err != nil {
			log.Printf("get: %v", err)
			return
		}
		if !ok {
			return
		}
		held = append(held, d)
		reason, count := deathInfo(d)
		fmt.Printf("%d) id=%s type=%s reason=%s dead-lettered=%dx body=%s\n",
			i+1, d.MessageId, d.Type, reason, count, d.Body)
	}
}

// move republishes messages to the main queue with a fresh retry counter,
// and only then removes them from the DLQ. If the republish fails the
// message stays in the DLQ, so nothing is lost.
func move(broker *messaging.RabbitMQ, queue, dlq string, limit int) {
	moved := 0
	for i := 0; i < limit; i++ {
		d, ok, err := broker.Get(dlq)
		if err != nil {
			log.Fatalf("get: %v", err)
		}
		if !ok {
			break
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = broker.Republish(ctx, queue, d,
			amqp.Table{messaging.RetryCountHeader: int32(0)})
		cancel()
		if err != nil {
			_ = d.Nack(false, true)
			log.Fatalf("replay %s failed, left in DLQ: %v", d.MessageId, err)
		}
		if err := d.Ack(false); err != nil {
			log.Fatalf("ack %s: %v", d.MessageId, err)
		}
		moved++
		fmt.Printf("replayed %s -> %s\n", d.MessageId, queue)
	}
	fmt.Printf("done: %d message(s) moved back\n", moved)
}

// deathInfo reads RabbitMQ's x-death header: why and how often the message
// was dead-lettered.
func deathInfo(d amqp.Delivery) (string, int64) {
	deaths, ok := d.Headers["x-death"].([]interface{})
	if !ok || len(deaths) == 0 {
		return "unknown", 0
	}
	first, ok := deaths[0].(amqp.Table)
	if !ok {
		return "unknown", 0
	}
	reason, _ := first["reason"].(string)
	count, _ := first["count"].(int64)
	return reason, count
}
