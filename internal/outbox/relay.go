package outbox

import (
	"context"
	"log/slog"
	"time"
)

const (
	// pollInterval is how often the relay looks for unsent events.
	pollInterval = 1 * time.Second

	// batchSize caps how many events one tick publishes.
	batchSize = 100

	// publishTimeout caps one publish + broker confirm.
	publishTimeout = 5 * time.Second
)

// Publisher sends one event to the broker and returns once it is stored.
// *messaging.RabbitMQ implements it.
type Publisher interface {
	PublishEvent(ctx context.Context, eventType, messageID string, body []byte) error
	Close() error
}

// ConnectFunc opens a new broker connection. The relay calls it lazily and
// again after a failure, so a broker restart heals by itself.
type ConnectFunc func() (Publisher, error)

// Relay moves events from the outbox table to RabbitMQ.
type Relay struct {
	store    Store
	connect  ConnectFunc
	pub      Publisher
	interval time.Duration
}

func NewRelay(store Store, connect ConnectFunc) *Relay {
	return &Relay{store: store, connect: connect, interval: pollInterval}
}

// Run polls until ctx is cancelled (graceful shutdown).
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	defer r.dropPublisher()

	slog.Info("Outbox relay started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox relay stopped")
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

// tick publishes one batch of unsent events.
func (r *Relay) tick(ctx context.Context) {
	if r.pub == nil {
		pub, err := r.connect()
		if err != nil {
			// Not fatal: events wait safely in the outbox until the broker is back.
			slog.Warn("Outbox relay: broker unavailable, events wait in outbox", "error", err.Error())
			return
		}
		r.pub = pub
		slog.Info("Outbox relay: connected to broker")
	}

	sent, failed, err := r.store.ProcessBatch(ctx, batchSize, func(m Message) error {
		pubCtx, cancel := context.WithTimeout(ctx, publishTimeout)
		defer cancel()
		return r.pub.PublishEvent(pubCtx, m.EventType, m.MessageID, m.Payload)
	})
	if err != nil {
		slog.Error("Outbox relay: batch failed", "error", err.Error())
		return
	}

	if sent > 0 {
		slog.Info("Outbox relay: events published", "count", sent)
	}
	if failed > 0 {
		// Drop the connection; the next tick reconnects.
		slog.Warn("Outbox relay: publish failed, will reconnect and retry")
		r.dropPublisher()
	}
}

func (r *Relay) dropPublisher() {
	if r.pub != nil {
		_ = r.pub.Close()
		r.pub = nil
	}
}
