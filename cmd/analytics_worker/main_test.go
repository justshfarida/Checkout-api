package main

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/analytics"
	"checkout-api/internal/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type fakeAck struct {
	acked, nacked, requeued bool
}

func (f *fakeAck) Ack(uint64, bool) error { f.acked = true; return nil }
func (f *fakeAck) Nack(_ uint64, _ bool, requeue bool) error {
	f.nacked, f.requeued = true, requeue
	return nil
}
func (f *fakeAck) Reject(_ uint64, requeue bool) error {
	f.nacked, f.requeued = true, requeue
	return nil
}

const validBody = `{"order_id":42,"total":1300,"paid_at":"2026-10-18T12:00:00Z"}`

func delivery(body, messageID string, retryCount int32) (amqp.Delivery, *fakeAck) {
	ack := &fakeAck{}
	d := amqp.Delivery{Acknowledger: ack, Body: []byte(body), MessageId: messageID}
	if retryCount > 0 {
		d.Headers = amqp.Table{messaging.RetryCountHeader: retryCount}
	}
	return d, ack
}

// fakeStore behaves like daily_sales + processed_messages for one day.
type fakeStore struct {
	seen    map[string]bool
	orders  int
	revenue int64
	err     error
}

func (s *fakeStore) record(_ context.Context, id string, e messaging.OrderPaidEvent) (analytics.DayTotals, bool, error) {
	if s.err != nil {
		return analytics.DayTotals{}, false, s.err
	}
	if s.seen[id] {
		return analytics.DayTotals{}, true, nil
	}
	s.seen[id] = true
	s.orders++
	s.revenue += int64(e.Total)
	return analytics.DayTotals{Day: "2026-10-18", Orders: s.orders, Revenue: s.revenue}, false, nil
}

func newWorker(store *fakeStore) (*worker, *[]int) {
	retries := &[]int{}
	return &worker{
		id:     "test",
		record: store.record,
		retry: func(_ context.Context, _ amqp.Delivery, attempt int) error {
			*retries = append(*retries, attempt)
			return nil
		},
	}, retries
}

func TestCountsOrderAndAcks(t *testing.T) {
	store := &fakeStore{seen: map[string]bool{}}
	w, _ := newWorker(store)
	msg, ack := delivery(validBody, "order.paid:42", 0)

	w.process(context.Background(), msg)

	if !ack.acked || store.orders != 1 || store.revenue != 1300 {
		t.Fatalf("acked=%v orders=%d revenue=%d", ack.acked, store.orders, store.revenue)
	}
}

func TestDuplicateIsNotCountedTwice(t *testing.T) {
	store := &fakeStore{seen: map[string]bool{}}
	w, _ := newWorker(store)

	first, _ := delivery(validBody, "order.paid:42", 0)
	w.process(context.Background(), first)
	second, ack := delivery(validBody, "order.paid:42", 0)
	w.process(context.Background(), second)

	if !ack.acked || store.orders != 1 {
		t.Fatalf("duplicate: acked=%v orders=%d, want acked and 1 order", ack.acked, store.orders)
	}
}

func TestInvalidMessageGoesToDLQ(t *testing.T) {
	store := &fakeStore{seen: map[string]bool{}}
	w, _ := newWorker(store)
	msg, ack := delivery(`{"order_id":"not-a-number"}`, "poison:1", 0)

	w.process(context.Background(), msg)

	if !ack.nacked || ack.requeued || store.orders != 0 {
		t.Fatalf("nacked=%v requeued=%v orders=%d", ack.nacked, ack.requeued, store.orders)
	}
}

func TestDatabaseErrorIsRetried(t *testing.T) {
	store := &fakeStore{seen: map[string]bool{}, err: errors.New("db down")}
	w, retries := newWorker(store)
	msg, ack := delivery(validBody, "order.paid:42", 0)

	w.process(context.Background(), msg)

	if !ack.acked || len(*retries) != 1 || (*retries)[0] != 1 {
		t.Fatalf("acked=%v retries=%v, want original acked and one retry with attempt 1", ack.acked, *retries)
	}
}

func TestGivesUpAfterMaxAttempts(t *testing.T) {
	store := &fakeStore{seen: map[string]bool{}, err: errors.New("db down")}
	w, retries := newWorker(store)
	msg, ack := delivery(validBody, "order.paid:42", maxAttempts-1)

	w.process(context.Background(), msg)

	if !ack.nacked || ack.requeued || len(*retries) != 0 {
		t.Fatalf("nacked=%v requeued=%v retries=%v, want dead-lettered", ack.nacked, ack.requeued, *retries)
	}
}
