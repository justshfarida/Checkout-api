package webhooklab

import (
	"context"
	"sync"
	"time"
)

// Deduper remembers which event ids were already received (idempotency).
// Two implementations: MemoryDeduper (task 3) and redisdedup.Deduper (task 5).
type Deduper interface {
	// FirstSeen records eventID and reports true only the first time.
	// Check and record are ONE atomic step, so two copies of the same event
	// arriving at the same moment cannot both get true.
	FirstSeen(ctx context.Context, eventID string) (bool, error)
}

// MemoryDeduper keeps seen ids in a sync.Map (task 3).
//
// sync.Map is a map that is safe for many goroutines at once (every HTTP
// request runs in its own goroutine). LoadOrStore is the atomic
// "check and record" step.
//
// Limits: it is lost on restart and not shared between two instances of
// the receiver, and it grows forever. That is why task 5 moves it to Redis.
type MemoryDeduper struct {
	seen sync.Map // event id -> time first seen
}

// FirstSeen implements Deduper.
func (m *MemoryDeduper) FirstSeen(_ context.Context, eventID string) (bool, error) {
	_, alreadyThere := m.seen.LoadOrStore(eventID, time.Now())
	return !alreadyThere, nil
}
