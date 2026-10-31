package webhooklab

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// HandlerFunc is one downstream consumer of an event.
type HandlerFunc func(ctx context.Context, e Event) error

// Handler is a named downstream consumer (fraud, email, analytics, ...).
type Handler struct {
	Name string
	Fn   HandlerFunc
}

// DeadLetterWriter stores events a handler gave up on (task 8).
type DeadLetterWriter interface {
	Append(v any) error
}

// DispatcherConfig tunes retries and buffering.
type DispatcherConfig struct {
	MaxAttempts    int           // tries per handler before the DLQ (default 3)
	RetryDelay     time.Duration // wait after the 1st failure, doubled each time
	QueueSize      int           // buffered events per handler (default 100)
	HandlerTimeout time.Duration // max time for one call (default 5s)
}

// ErrDispatcherClosed is returned by Dispatch after Close.
var ErrDispatcherClosed = errors.New("dispatcher is closed")

// Dispatcher is the fan-out (task 7): every event is copied to one channel
// per handler, and each handler reads its own channel in its own goroutine.
// So a slow or failing handler never delays the others, like the separate
// queues of a RabbitMQ fanout exchange, but inside one process.
//
//	Dispatch(e) --+--> chan --> goroutine: fraud
//	              +--> chan --> goroutine: email
//	              +--> chan --> goroutine: analytics
type Dispatcher struct {
	handlers []Handler
	queues   []chan Event
	dlq      DeadLetterWriter
	cfg      DispatcherConfig

	mu     sync.RWMutex // guards closed: sending on a closed channel panics
	closed bool
	wg     sync.WaitGroup
}

// NewDispatcher starts one goroutine per handler.
func NewDispatcher(handlers []Handler, dlq DeadLetterWriter, cfg DispatcherConfig) *Dispatcher {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.HandlerTimeout <= 0 {
		cfg.HandlerTimeout = 5 * time.Second
	}

	d := &Dispatcher{handlers: handlers, dlq: dlq, cfg: cfg}
	for _, h := range handlers {
		q := make(chan Event, cfg.QueueSize)
		d.queues = append(d.queues, q)
		d.wg.Add(1)
		go d.run(h, q)
	}
	return d
}

// Dispatch hands the event to every handler and returns immediately; the
// work happens in the handler goroutines.
func (d *Dispatcher) Dispatch(e Event) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		return ErrDispatcherClosed
	}
	for _, q := range d.queues {
		q <- e // blocks only if that handler is QueueSize events behind
	}
	return nil
}

// Close stops accepting events, lets every handler finish what is already
// in its channel, and waits for them (graceful shutdown).
func (d *Dispatcher) Close() {
	d.mu.Lock()
	if !d.closed {
		d.closed = true
		for _, q := range d.queues {
			close(q) // ends the "for range" loop in run once it is empty
		}
	}
	d.mu.Unlock()
	d.wg.Wait()
}

func (d *Dispatcher) run(h Handler, q <-chan Event) {
	defer d.wg.Done()
	for e := range q {
		d.deliver(h, e)
	}
}

// deliver calls the handler up to MaxAttempts times with a growing pause,
// then writes the event to the dead-letter file (task 8).
func (d *Dispatcher) deliver(h Handler, e Event) {
	var err error
	for attempt := 1; attempt <= d.cfg.MaxAttempts; attempt++ {
		if err = d.call(h, e); err == nil {
			return
		}
		log.Printf("[%s] %s: attempt %d/%d failed: %v", h.Name, e.EventID, attempt, d.cfg.MaxAttempts, err)
		if attempt < d.cfg.MaxAttempts {
			time.Sleep(d.cfg.RetryDelay << (attempt - 1)) // 1x, 2x, 4x ...
		}
	}

	dl := DeadLetter{
		FailedAt: time.Now().UTC(),
		Handler:  h.Name,
		Attempts: d.cfg.MaxAttempts,
		Error:    err.Error(),
		Event:    e,
	}
	if werr := d.dlq.Append(dl); werr != nil {
		log.Printf("[%s] %s: could not write dead letter: %v", h.Name, e.EventID, werr)
		return
	}
	log.Printf("[%s] %s: giving up -> dead_letter.jsonl", h.Name, e.EventID)
}

// call runs one attempt with a timeout. A panicking handler counts as a
// failed attempt instead of crashing the whole receiver.
func (d *Dispatcher) call(h Handler, e Event) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.HandlerTimeout)
	defer cancel()
	return h.Fn(ctx, e)
}
