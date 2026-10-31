package messaging

// Package messaging connects the API to RabbitMQ: it declares the queues
// and publishes events. Workers use the same package to consume.
import (
	"context"
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

	// OrderEventsExchange is a fanout exchange (exercise 4): every event
	// published here is copied into EVERY queue bound to it, so each
	// consumer (email, analytics, ...) gets its own copy.
	OrderEventsExchange = "order.events"

	// OrderAnalyticsQueue holds the analytics worker's copy of each event.
	OrderAnalyticsQueue = "order_analytics"

	// OrderAnalyticsDLQ is the analytics queue's own dead-letter queue.
	OrderAnalyticsDLQ = "order_analytics.dlq"
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
//
//	                         +--> order_notifications --> email worker
//	outbox relay --> order.events (fanout)
//	                         +--> order_analytics     --> analytics worker
//
// Each queue dead-letters into "dlx", which routes to that queue's own DLQ.
func (r *RabbitMQ) declareTopology() error {
	// 1. Dead-letter exchange: routes rejected messages by exact routing key.
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

	// 2. Fanout exchange for order events (exercise 4).
	if err := r.ch.ExchangeDeclare(
		OrderEventsExchange,
		"fanout", // copy every message to all bound queues, ignore routing key
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	); err != nil {
		return fmt.Errorf("declare order events exchange: %w", err)
	}

	// 3. One queue per consumer, each with its own DLQ, each bound to the
	// fanout exchange. Adding a new consumer later = one more line here.
	for _, q := range []struct{ name, dlq string }{
		{OrderNotificationsQueue, OrderNotificationsDLQ},
		{OrderAnalyticsQueue, OrderAnalyticsDLQ},
	} {
		if err := r.declareQueueWithDLQ(q.name, q.dlq); err != nil {
			return err
		}
		if err := r.ch.QueueBind(
			q.name,              // queue
			"",                  // routing key: ignored by fanout exchanges
			OrderEventsExchange, // exchange
			false,               // noWait
			nil,                 // args
		); err != nil {
			return fmt.Errorf("bind %s to %s: %w", q.name, OrderEventsExchange, err)
		}
	}

	return nil
}

// declareQueueWithDLQ declares a durable queue whose rejected messages go to
// "dlx" with routing key dlq, plus the DLQ itself bound to "dlx".
//
// Careful: the queue arguments must stay exactly the same once a queue
// exists, or RabbitMQ refuses the declare with PRECONDITION_FAILED.
func (r *RabbitMQ) declareQueueWithDLQ(queue, dlq string) error {
	if _, err := r.ch.QueueDeclare(
		dlq,   // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	); err != nil {
		return fmt.Errorf("declare dead-letter queue %s: %w", dlq, err)
	}

	if err := r.ch.QueueBind(
		dlq,                // queue
		dlq,                // routing key
		DeadLetterExchange, // exchange
		false,              // noWait
		nil,                // args
	); err != nil {
		return fmt.Errorf("bind dead-letter queue %s: %w", dlq, err)
	}

	// When a message here is rejected without requeue, RabbitMQ sends it to
	// the "dlx" exchange with routing key <dlq> -> it lands in the DLQ.
	if _, err := r.ch.QueueDeclare(
		queue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		amqp.Table{
			"x-dead-letter-exchange":    DeadLetterExchange,
			"x-dead-letter-routing-key": dlq,
		},
	); err != nil {
		return fmt.Errorf("declare queue %s: %w", queue, err)
	}
	return nil
}

// PublishEvent publishes one event to the order.events fanout exchange, so
// every consumer queue gets a copy, and waits until RabbitMQ confirms it
// has stored the message (publisher confirms). Only then may the caller
// treat the event as sent.
func (r *RabbitMQ) PublishEvent(ctx context.Context, eventType, messageID string, body []byte) error {
	r.mu.Lock()
	confirm, err := r.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		OrderEventsExchange, // exchange: fanout, copies to all bound queues
		eventType,           // routing key: ignored by fanout, kept for readability
		false,               // mandatory
		false,               // immediate
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

// RetryCountHeader counts how many times a worker already retried a message
// (exercise 3). After maxAttempts in the worker the message goes to the DLQ.
const RetryCountHeader = "x-retry-count"

// Republish puts a copy of a delivered message back on ONE queue (default
// exchange, not the fanout), with extra headers, and waits for the broker to
// confirm it. Only the consumer that failed gets the retry; the others
// already handled their own copy. Used by the worker to
// retry a message with a counter, and by dlq-replay to move a message from
// the DLQ back to the main queue.
func (r *RabbitMQ) Republish(ctx context.Context, queue string, d amqp.Delivery, headers amqp.Table) error {
	merged := amqp.Table{}
	for k, v := range d.Headers {
		// x-death is RabbitMQ's dead-letter history; a fresh copy starts clean.
		if k == "x-death" {
			continue
		}
		merged[k] = v
	}
	for k, v := range headers {
		merged[k] = v
	}

	r.mu.Lock()
	confirm, err := r.ch.PublishWithDeferredConfirmWithContext(
		ctx, "", queue, false, false,
		amqp.Publishing{
			MessageId:    d.MessageId,
			Type:         d.Type,
			ContentType:  d.ContentType,
			DeliveryMode: amqp.Persistent,
			Timestamp:    d.Timestamp,
			Headers:      merged,
			Body:         d.Body,
		},
	)
	r.mu.Unlock()
	if err != nil {
		return fmt.Errorf("republish %s: %w", d.MessageId, err)
	}

	stored, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("confirm republish %s: %w", d.MessageId, err)
	}
	if !stored {
		return fmt.Errorf("broker refused republish of %s", d.MessageId)
	}
	return nil
}

// Get pulls one message from a queue without waiting (ok=false when empty).
// The caller must Ack or Nack it. Used by the dlq-replay tool.
func (r *RabbitMQ) Get(queue string) (amqp.Delivery, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ch.Get(queue, false)
}

// QueueDepth returns how many messages are ready in a queue.
func (r *RabbitMQ) QueueDepth(queue string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, err := r.ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	if err != nil {
		return 0, fmt.Errorf("inspect %s: %w", queue, err)
	}
	return q.Messages, nil
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
