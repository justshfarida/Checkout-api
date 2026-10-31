// Package notifier sends customer notifications. Email is simulated: a real
// provider (SendGrid, SES) would be called here.
package notifier

import (
	"context"
	"log"
	"time"

	"checkout-api/internal/messaging"
)

// emailSendDuration models a real email provider's latency.
const emailSendDuration = 2 * time.Second

// SendOrderConfirmation "sends" the confirmation email for a paid order.
// It respects ctx so a shutting-down worker isn't stuck in it.
func SendOrderConfirmation(ctx context.Context, event messaging.OrderPaidEvent) error {
	select {
	case <-time.After(emailSendDuration):
	case <-ctx.Done():
		return ctx.Err()
	}

	log.Printf("[EMAIL] order #%d confirmed, total %d", event.OrderID, event.Total)
	return nil
}
