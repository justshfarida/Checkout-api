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
// It is written into the outbox table by the payment transaction (see
// xsolla_repository.go) and published from there by the outbox relay.
type OrderPaidEvent struct {
	OrderID int       `json:"order_id"`
	UserID  int       `json:"user_id"`
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

	// Confirm mode: the broker acknowledges every published message once it
	// has stored it, so the outbox relay knows when an event is really sent.
	if err := ch.Confirm(false); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
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

// PublishEvent publishes one event to the order_notifications queue and
// waits until RabbitMQ confirms it has stored the message (publisher
// confirms). Only then may the caller treat the event as sent.
func (r *RabbitMQ) PublishEvent(ctx context.Context, eventType, messageID string, body []byte) error {
	r.mu.Lock()
	confirm, err := r.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		"",                      // exchange: "" = default exchange
		OrderNotificationsQueue, // routing key = queue name (default exchange rule)
		false,                   // mandatory
		false,                   // immediate
		amqp.Publishing{
			// Stable id from the outbox, e.g. "order.paid:42": lets the
			// consumer recognise a duplicate delivery (exercise 1).
			MessageId:    messageID,
			Type:         eventType,
			DeliveryMode: amqp.Persistent, // written to disk, survives a broker restart
			ContentType:  "application/json",
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	r.mu.Unlock()
	if err != nil {
		return fmt.Errorf("publish %s: %w", messageID, err)
	}

	// Wait for the broker's answer: ack = stored, nack = refused.
	stored, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("confirm %s: %w", messageID, err)
	}
	if !stored {
		return fmt.Errorf("broker refused %s", messageID)
	}
	return nil
}

// Consume starts receiving messages from a queue. Messages are NOT
// auto-acknowledged: the caller must Ack or Nack each one, so a worker
// that crashes mid-job gets its message redelivered instead of lost.
func (r *RabbitMQ) Consume(queue, consumerTag string, prefetch int) (<-chan amqp.Delivery, error) {
	// Prefetch: give this worker at most N unacknowledged messages at a time.
	if err := r.ch.Qos(prefetch, 0, false); err != nil {
		return nil, fmt.Errorf("set prefetch: %w", err)
	}

	deliveries, err := r.ch.Consume(
		queue,
		consumerTag, // shown in the RabbitMQ UI, e.g. the worker id
		false,       // autoAck: false = we ack manually after the work is done
		false,       // exclusive: false = several workers may share the queue
		false,       // noLocal (not supported by RabbitMQ)
		false,       // noWait
		nil,         // args
	)
	if err != nil {
		return nil, fmt.Errorf("start consuming %s: %w", queue, err)
	}
	return deliveries, nil
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
