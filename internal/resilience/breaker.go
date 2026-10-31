// Package resilience contains small, dependency-free resilience patterns.
package resilience

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by Allow when calls are being rejected
// without trying the protected dependency.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State is the current mode of a Breaker.
type State int

const (
	// Closed: calls go through; consecutive failures are counted.
	Closed State = iota
	// Open: calls are rejected immediately until the cooldown has passed.
	Open
	// HalfOpen: one trial call is let through to test if the dependency recovered.
	HalfOpen
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Breaker is a hand-rolled, three-state circuit breaker.
//
//	Closed --(threshold consecutive failures)--> Open
//	Open --(cooldown elapsed, next call)--> HalfOpen (one trial call)
//	HalfOpen --(trial succeeds)--> Closed
//	HalfOpen --(trial fails)--> Open
//
// It is safe for concurrent use.
type Breaker struct {
	name      string
	threshold int
	cooldown  time.Duration
	now       func() time.Time

	mu            sync.Mutex
	state         State
	failures      int
	openedAt      time.Time
	trialInFlight bool
}

// NewBreaker creates a breaker that opens after `threshold` consecutive
// failures and stays open for `cooldown` before allowing a trial call.
func NewBreaker(name string, threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{
		name:      name,
		threshold: threshold,
		cooldown:  cooldown,
		now:       time.Now,
		state:     Closed,
	}
}

// Allow reports whether a call may proceed. It returns ErrCircuitOpen when
// the call must fail fast. Every allowed call must be followed by exactly
// one of Success, Failure or Cancel.
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case Open:
		if b.now().Sub(b.openedAt) < b.cooldown {
			return ErrCircuitOpen
		}
		//if 30 secs have passed
		b.setState(HalfOpen)
		b.trialInFlight = true //one probe request is running
		return nil

	case HalfOpen:
		// Only one trial call at a time; everyone else keeps failing fast.
		if b.trialInFlight {
			return ErrCircuitOpen
		}
		b.trialInFlight = true
		return nil

	default: // Closed
		return nil
	}
}

// Success records a successful call and closes the breaker.
func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures = 0
	b.trialInFlight = false
	if b.state != Closed {
		b.setState(Closed)
	}
}

// Failure records a failed call. In Closed state it opens the breaker after
// `threshold` consecutive failures; a failed trial in HalfOpen reopens it.
func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case HalfOpen:
		b.trialInFlight = false
		b.open()

	case Closed:
		b.failures++
		if b.failures >= b.threshold {
			b.open()
		}
	}
}

// Cancel releases an allowed call that ended without a verdict (for example
// the client disconnected), so a half-open trial slot is not stuck forever.
func (b *Breaker) Cancel() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.trialInFlight = false
}

// State returns the current state (for logs, metrics and tests).
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.state
}

func (b *Breaker) open() {
	b.openedAt = b.now()
	b.failures = 0
	b.setState(Open)
}

func (b *Breaker) setState(s State) {
	from := b.state
	b.state = s
	slog.Warn("Circuit breaker state changed",
		"breaker", b.name,
		"from", from.String(),
		"to", s.String(),
	)
}
