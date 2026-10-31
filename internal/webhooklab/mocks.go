package webhooklab

import (
	"context"
	"errors"
	"log"
	"time"
)

// ErrSimulatedOutage is what a handler returns when it is set to fail.
var ErrSimulatedOutage = errors.New("simulated outage")

// MockHandlers returns the three mock downstream services from the lecture
// (slide 37). The one named failing always fails, to demo the DLQ.
// work is how long each one pretends to work.
func MockHandlers(failing string, work time.Duration) []Handler {
	mock := func(name, action string) Handler {
		return Handler{Name: name, Fn: func(ctx context.Context, e Event) error {
			select {
			case <-time.After(work):
			case <-ctx.Done():
				return ctx.Err()
			}
			if name == failing {
				return ErrSimulatedOutage
			}
			log.Printf("[%s] %s %s (%s)", name, action, e.EventID, e.Event)
			return nil
		}}
	}
	return []Handler{
		mock("fraud", "checked"),
		mock("email", "sent receipt for"),
		mock("analytics", "counted"),
	}
}
