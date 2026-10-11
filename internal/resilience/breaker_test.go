package resilience

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// newTestBreaker returns a breaker with a controllable clock.
func newTestBreaker(threshold int, cooldown time.Duration) (*Breaker, *time.Time) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := NewBreaker("test", threshold, cooldown)
	b.now = func() time.Time { return now }
	return b, &now
}

func failN(b *Breaker, n int) {
	for i := 0; i < n; i++ {
		if b.Allow() == nil {
			b.Failure()
		}
	}
}

func TestBreaker_StaysClosedBelowThreshold(t *testing.T) {
	b, _ := newTestBreaker(3, time.Minute)

	failN(b, 2)

	if b.State() != Closed {
		t.Fatalf("state = %v, want closed after 2 of 3 failures", b.State())
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() = %v, want nil while closed", err)
	}
}

func TestBreaker_SuccessResetsFailureCount(t *testing.T) {
	b, _ := newTestBreaker(3, time.Minute)

	failN(b, 2)
	_ = b.Allow()
	b.Success()
	failN(b, 2)

	if b.State() != Closed {
		t.Fatalf("state = %v; failures must be consecutive to open", b.State())
	}
}

func TestBreaker_OpensAfterThresholdAndFailsFast(t *testing.T) {
	b, _ := newTestBreaker(3, time.Minute)

	failN(b, 3)

	if b.State() != Open {
		t.Fatalf("state = %v, want open", b.State())
	}
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow() = %v, want ErrCircuitOpen during cooldown", err)
	}
}

func TestBreaker_HalfOpenAfterCooldownAllowsOneTrial(t *testing.T) {
	b, now := newTestBreaker(1, 30*time.Second)
	failN(b, 1)

	*now = now.Add(30 * time.Second)

	if err := b.Allow(); err != nil {
		t.Fatalf("first call after cooldown: Allow() = %v, want nil (trial)", err)
	}
	if b.State() != HalfOpen {
		t.Fatalf("state = %v, want half-open", b.State())
	}
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("second concurrent call: Allow() = %v, want ErrCircuitOpen", err)
	}
}

func TestBreaker_TrialSuccessCloses(t *testing.T) {
	b, now := newTestBreaker(1, time.Second)
	failN(b, 1)
	*now = now.Add(time.Second)

	_ = b.Allow()
	b.Success()

	if b.State() != Closed {
		t.Fatalf("state = %v, want closed after successful trial", b.State())
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() = %v, want nil once closed", err)
	}
}

func TestBreaker_TrialFailureReopensWithNewCooldown(t *testing.T) {
	b, now := newTestBreaker(1, 10*time.Second)
	failN(b, 1)
	*now = now.Add(10 * time.Second)

	_ = b.Allow()
	b.Failure()

	if b.State() != Open {
		t.Fatalf("state = %v, want open after failed trial", b.State())
	}
	*now = now.Add(5 * time.Second)
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow() = %v, want ErrCircuitOpen: cooldown restarts after a failed trial", err)
	}
}

func TestBreaker_CancelFreesTheTrialSlot(t *testing.T) {
	b, now := newTestBreaker(1, time.Second)
	failN(b, 1)
	*now = now.Add(time.Second)

	_ = b.Allow() // trial started...
	b.Cancel()    // ...but the client went away without a verdict

	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() = %v, want a new trial to be allowed after Cancel", err)
	}
}

func TestBreaker_ConcurrentUseIsSafe(t *testing.T) {
	b := NewBreaker("race", 1000, time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if b.Allow() == nil {
				if i%2 == 0 {
					b.Success()
				} else {
					b.Failure()
				}
			}
		}(i)
	}
	wg.Wait()
}
