package messaging

// Package messaging connects the API to RabbitMQ: it declares the queues
// and publishes events. Workers use the same package to consume.
import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// OrderNotificationsQueue holds "order paid" jobs for the email worker.
	OrderNotificationsQueue = "order_notifications"

	// DeadLetterExchange receives messages a worker rejected for good.
	DeadLetterExchange = "dlx"

	// OrderNotificationsDLQ is where rejected (poison) messages end up,
	// so they can be inspected and replayed instead of being lost.
	OrderNotificationsDLQ = "order_notifications.dlq"
)

// OrderPaidEvent is the message body: just enough for a worker to do its job.
type OrderPaidEvent struct {
	OrderID int       `json:"order_id"`
	Total   int       `json:"total"`
	PaidAt  time.Time `json:"paid_at"`
}

// RabbitMQ holds one connection and one channel to the broker.
type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel

	// An amqp channel must not be used by two goroutines at the same time,
	// and many HTTP requests may publish at once, so publishing is locked.
	mu sync.Mutex
}

// Connect opens the connection, opens a channel and declares the topology.
func Connect(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	r := &RabbitMQ{conn: conn, ch: ch}

	if err := r.declareTopology(); err != nil {
		r.Close()
		return nil, err
	}

	return r, nil
}

// declareTopology creates the exchanges, queues and bindings up front.
// Declaring is idempotent: if they already exist with the same settings,
// nothing changes, so this is safe on every startup.
func (r *RabbitMQ) declareTopology() error {
	// 1. Dead-letter exchange + dead-letter queue, bound together.
	if err := r.ch.ExchangeDeclare(
		DeadLetterExchange, // name
		"direct",           // type: route by exact routing key
		true,               // durable: survives a broker restart
		false,              // autoDelete
		false,              // internal
		false,              // noWait
		nil,                // args
	); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}

	if _, err := r.ch.QueueDeclare(
		OrderNotificationsDLQ, // name
		true,                  // durable
		false,                 // autoDelete
		false,                 // exclusive
		false,                 // noWait
		nil,                   // args
	); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}

	if err := r.ch.QueueBind(
		OrderNotificationsDLQ, // queue
		OrderNotificationsDLQ, // routing key
		DeadLetterExchange,    // exchange
		false,                 // noWait
		nil,                   // args
	); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}

	// 2. Main queue. Its arguments say: when a message here is rejected
	// without requeue, send it to the "dlx" exchange with routing key
	// "order_notifications.dlq" -> it lands in the DLQ.
	if _, err := r.ch.QueueDeclare(
		OrderNotificationsQueue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		amqp.Table{
			"x-dead-letter-exchange":    DeadLetterExchange,
			"x-dead-letter-routing-key": OrderNotificationsDLQ,
		},
	); err != nil {
		return fmt.Errorf("declare order notifications queue: %w", err)
	}

	return nil
}

// PublishOrderPaid sends one "order paid" job to the order_notifications queue.
func (r *RabbitMQ) PublishOrderPaid(ctx context.Context, event OrderPaidEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode order paid event: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ch.PublishWithContext(
		ctx,
		"",                      // exchange: "" = default exchange
		OrderNotificationsQueue, // routing key = queue name (default exchange rule)
		false,                   // mandatory
		false,                   // immediate
		amqp.Publishing{
			// Same order -> same ID, every time. If this event is ever
			// delivered twice, the worker can recognise the duplicate
			// (exercise 1: idempotent consumer).
			MessageId:    fmt.Sprintf("order.paid:%d", event.OrderID),
			Type:         "order.paid",
			DeliveryMode: amqp.Persistent, // written to disk, survives a broker restart
			ContentType:  "application/json",
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
}

// Close closes the channel and the connection.
func (r *RabbitMQ) Close() error {
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
