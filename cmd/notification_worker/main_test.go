package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"checkout-api/internal/messaging"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ---------- fakes ----------

// fakeAck records what the worker answered for one delivery.
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

// fakeDB behaves like the processed_messages table: key -> processed?
type fakeDB struct {
	rows map[string]bool
	err  error
}

func newFakeDB() *fakeDB { return &fakeDB{rows: map[string]bool{}} }

func key(args []any) string { return args[0].(string) + "|" + args[1].(string) }

func (f *fakeDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.err != nil {
		return pgconn.CommandTag{}, f.err
	}
	k := key(args)
	if strings.HasPrefix(strings.TrimSpace(sql), "INSERT") {
		if _, exists := f.rows[k]; exists {
			return pgconn.NewCommandTag("INSERT 0 0"), nil // ON CONFLICT DO NOTHING
		}
		f.rows[k] = false
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	f.rows[k] = true // UPDATE ... SET processed_at = now()
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (f *fakeDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	return fakeRow{done: f.rows[key(args)]}
}

type fakeRow struct{ done bool }

func (r fakeRow) Scan(dest ...any) error {
	*(dest[0].(*bool)) = r.done
	return nil
}

// ---------- helpers ----------

const validBody = `{"order_id":42,"total":1300,"paid_at":"2026-10-18T12:00:00Z"}`

func delivery(body, messageID string) (amqp.Delivery, *fakeAck) {
	ack := &fakeAck{}
	return amqp.Delivery{Acknowledger: ack, Body: []byte(body), MessageId: messageID}, ack
}

// newWorker returns a worker whose email sender counts calls and can fail.
func newWorker(db messaging.DB, emailErr *error) (*worker, *int) {
	sent := 0
	w := &worker{
		id: "test",
		db: db,
		sendEmail: func(context.Context, messaging.OrderPaidEvent) error {
			if emailErr != nil && *emailErr != nil {
				return *emailErr
			}
			sent++
			return nil
		},
		requeueDelay: 0,
	}
	return w, &sent
}

// ---------- tests ----------

func TestProcess_FirstDeliverySendsEmailAndAcks(t *testing.T) {
	db := newFakeDB()
	w, sent := newWorker(db, nil)
	msg, ack := delivery(validBody, "order.paid:42")

	w.process(context.Background(), msg)

	if *sent != 1 || !ack.acked {
		t.Fatalf("sent=%d acked=%v, want 1 email and Ack", *sent, ack.acked)
	}
	if !db.rows["order.paid|order.paid:42"] {
		t.Error("message not marked processed")
	}
}

func TestProcess_DuplicateDeliveryIsSkipped(t *testing.T) {
	db := newFakeDB()
	w, sent := newWorker(db, nil)

	first, _ := delivery(validBody, "order.paid:42")
	w.process(context.Background(), first)

	second, ack := delivery(validBody, "order.paid:42") // same MessageId = redelivery
	w.process(context.Background(), second)

	if *sent != 1 {
		t.Fatalf("emails sent = %d, want 1 (duplicate must be skipped)", *sent)
	}
	if !ack.acked {
		t.Error("duplicate must be acked so it leaves the queue")
	}
}

func TestProcess_InvalidMessageGoesToDLQ(t *testing.T) {
	for name, tc := range map[string]struct{ body, id string }{
		"broken JSON":   {`{not json`, "order.paid:1"},
		"no order id":   {`{"total":5}`, "order.paid:1"},
		"no message id": {validBody, ""},
	} {
		t.Run(name, func(t *testing.T) {
			w, sent := newWorker(newFakeDB(), nil)
			msg, ack := delivery(tc.body, tc.id)

			w.process(context.Background(), msg)

			if !ack.nacked || ack.requeued || *sent != 0 {
				t.Fatalf("nacked=%v requeued=%v sent=%d, want Nack(requeue=false) and no email",
					ack.nacked, ack.requeued, *sent)
			}
		})
	}
}

func TestProcess_EmailFailureIsRequeuedAndRetriedLater(t *testing.T) {
	db := newFakeDB()
	emailErr := errors.New("smtp down")
	w, sent := newWorker(db, &emailErr)

	msg, ack := delivery(validBody, "order.paid:42")
	w.process(context.Background(), msg)

	if !ack.nacked || !ack.requeued {
		t.Fatalf("nacked=%v requeued=%v, want Nack(requeue=true)", ack.nacked, ack.requeued)
	}

	// Provider is back: the redelivered message must still send the email,
	// because the first attempt never finished.
	emailErr = nil
	retry, retryAck := delivery(validBody, "order.paid:42")
	w.process(context.Background(), retry)

	if *sent != 1 || !retryAck.acked {
		t.Fatalf("after retry sent=%d acked=%v, want 1 email and Ack", *sent, retryAck.acked)
	}
}

func TestProcess_DatabaseErrorIsRequeuedWithoutEmail(t *testing.T) {
	db := newFakeDB()
	db.err = errors.New("db down")
	w, sent := newWorker(db, nil)
	msg, ack := delivery(validBody, "order.paid:42")

	w.process(context.Background(), msg)

	if !ack.requeued || *sent != 0 {
		t.Fatalf("requeued=%v sent=%d, want requeue and no email", ack.requeued, *sent)
	}
}
