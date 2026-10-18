package outbox

import (
	"context"
	"errors"
	"testing"
)

// fakeStore keeps outbox rows in memory and mimics ProcessBatch.
type fakeStore struct {
	rows []Message
	sent map[int64]bool
}

func newFakeStore(ids ...int64) *fakeStore {
	s := &fakeStore{sent: map[int64]bool{}}
	for _, id := range ids {
		s.rows = append(s.rows, Message{ID: id, EventType: "order.paid", MessageID: "order.paid:x", Payload: []byte(`{}`)})
	}
	return s
}

func (s *fakeStore) ProcessBatch(_ context.Context, limit int, publish func(Message) error) (int, int, error) {
	sent, failed := 0, 0
	for _, m := range s.rows {
		if s.sent[m.ID] || sent+failed >= limit {
			continue
		}
		if err := publish(m); err != nil {
			failed++
			break
		}
		s.sent[m.ID] = true
		sent++
	}
	return sent, failed, nil
}

// fakePublisher records published events and can fail.
type fakePublisher struct {
	err       error
	published []string
	closed    bool
}

func (p *fakePublisher) PublishEvent(_ context.Context, _, messageID string, _ []byte) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, messageID)
	return nil
}

func (p *fakePublisher) Close() error { p.closed = true; return nil }

func TestRelay_PublishesUnsentEventsAndMarksThemSent(t *testing.T) {
	store := newFakeStore(1, 2, 3)
	pub := &fakePublisher{}
	r := NewRelay(store, func() (Publisher, error) { return pub, nil })

	r.tick(context.Background())

	if len(pub.published) != 3 {
		t.Fatalf("published %d events, want 3", len(pub.published))
	}
	for _, id := range []int64{1, 2, 3} {
		if !store.sent[id] {
			t.Errorf("row %d not marked sent", id)
		}
	}

	// Next tick: nothing left to send.
	r.tick(context.Background())
	if len(pub.published) != 3 {
		t.Errorf("published again: %d events, want still 3", len(pub.published))
	}
}

func TestRelay_BrokerDownKeepsEventsInOutbox(t *testing.T) {
	store := newFakeStore(1)
	r := NewRelay(store, func() (Publisher, error) { return nil, errors.New("connection refused") })

	r.tick(context.Background())

	if store.sent[1] {
		t.Fatal("event marked sent although the broker was down")
	}
}

func TestRelay_PublishFailureRetriesAfterReconnect(t *testing.T) {
	store := newFakeStore(1)
	broken := &fakePublisher{err: errors.New("channel closed")}
	healthy := &fakePublisher{}
	connects := 0
	r := NewRelay(store, func() (Publisher, error) {
		connects++
		if connects == 1 {
			return broken, nil
		}
		return healthy, nil
	})

	r.tick(context.Background()) // publish fails -> connection dropped
	if store.sent[1] {
		t.Fatal("failed publish must not mark the row sent")
	}
	if !broken.closed {
		t.Error("broken connection should be closed")
	}

	r.tick(context.Background()) // reconnects -> succeeds
	if !store.sent[1] || len(healthy.published) != 1 {
		t.Fatalf("after reconnect sent=%v published=%d, want the event delivered once",
			store.sent[1], len(healthy.published))
	}
}
