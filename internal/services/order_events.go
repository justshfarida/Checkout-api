package services

import (
	"context"

	"checkout-api/internal/messaging"
)

// OrderEventPublisher is the "port" the order service uses to announce that
// something happened to an order. The service does not know about RabbitMQ;
// main.go plugs in the real implementation.
type OrderEventPublisher interface {
	PublishOrderPaid(ctx context.Context, event messaging.OrderPaidEvent) error
}

// NoopPublisher does nothing. It is the default, so tests and setups without
// a broker work without nil checks.
type NoopPublisher struct{}

func (NoopPublisher) PublishOrderPaid(context.Context, messaging.OrderPaidEvent) error {
	return nil
}
