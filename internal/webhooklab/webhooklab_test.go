package webhooklab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testSecret = "test-secret"

// ---------- task 2: HMAC ----------

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"event":"payment.completed","event_id":"evt_1"}`)
	good := Sign(testSecret, body)

	cases := map[string]struct {
		sig  string
		want bool
	}{
		"correct":          {good, true},
		"github prefix":    {"sha256=" + good, true},
		"missing":          {"", false},
		"wrong secret":     {Sign("other-secret", body), false},
		"not hex":          {"nonsense", false},
		"body changed one": {Sign(testSecret, append([]byte{}, body[:len(body)-1]...)), false},
	}
	for name, c := range cases {
		if got := Verify(testSecret, body, c.sig); got != c.want {
			t.Errorf("%s: Verify = %v, want %v", name, got, c.want)
		}
	}
}

// ---------- task 3: sync.Map dedup ----------

func TestMemoryDeduperConcurrent(t *testing.T) {
	var d MemoryDeduper
	var firsts atomic.Int32
	var wg sync.WaitGroup
	for range 50 { // 50 copies of the same event at the same moment
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := d.FirstSeen(context.Background(), "evt_1"); ok {
				firsts.Add(1)
			}
		}()
	}
	wg.Wait()
	if firsts.Load() != 1 {
		t.Fatalf("FirstSeen returned true %d times, want exactly 1", firsts.Load())
	}
	if ok, _ := d.FirstSeen(context.Background(), "evt_2"); !ok {
		t.Fatal("a new id must be first")
	}
}

// ---------- receiver (tasks 1, 2, 3, 6) ----------

type testReceiver struct {
	rc  *Receiver
	srv http.Handler
	mu  sync.Mutex
	got []Event
	log string
}

func newTestReceiver(t *testing.T) *testReceiver {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.jsonl")
	f, err := OpenJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })

	tr := &testReceiver{log: path}
	tr.rc = &Receiver{
		Secret: testSecret,
		Log:    f,
		Dedup:  &MemoryDeduper{},
		Dispatch: func(e Event) error {
			tr.mu.Lock()
			defer tr.mu.Unlock()
			tr.got = append(tr.got, e)
			return nil
		},
	}
	tr.srv = tr.rc.Routes()
	return tr
}

func (tr *testReceiver) post(path string, body []byte, sig string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if sig != "" {
		req.Header.Set(SignatureHeader, sig)
	}
	if len(header) == 2 {
		req.Header.Set(header[0], header[1])
	}
	rec := httptest.NewRecorder()
	tr.srv.ServeHTTP(rec, req)
	return rec
}

func (tr *testReceiver) dispatched() int {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return len(tr.got)
}

func eventBody(id string) []byte {
	return []byte(`{"event":"payment.completed","event_id":"` + id + `","timestamp":"2026-10-18T10:00:00Z",
  "data":{"order_id":"order_456","amount":99.99}}`) // multi-line on purpose
}

func TestWebhookAcceptsSignedEventAndLogsIt(t *testing.T) {
	tr := newTestReceiver(t)
	body := eventBody("evt_1")

	rec := tr.post("/webhook", body, Sign(testSecret, body))

	if rec.Code != http.StatusOK || tr.dispatched() != 1 {
		t.Fatalf("status %d, dispatched %d; want 200 and 1", rec.Code, tr.dispatched())
	}
	entries, err := ReadEventLog(tr.log)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].EventID != "evt_1" || entries[0].ReceivedAt.IsZero() {
		t.Fatalf("log = %+v, want one evt_1 entry with a timestamp", entries)
	}
	raw, _ := os.ReadFile(tr.log)
	if bytes.Count(raw, []byte("\n")) != 1 {
		t.Fatalf("a multi-line body must still be one line in the log:\n%s", raw)
	}
}

func TestWebhookRejectsMissingOrBadSignature(t *testing.T) {
	tr := newTestReceiver(t)
	body := eventBody("evt_1")

	for name, sig := range map[string]string{
		"missing": "",
		"wrong":   Sign("attacker-guess", body),
	} {
		rec := tr.post("/webhook", body, sig)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s signature: status %d, want 401", name, rec.Code)
		}
	}
	if tr.dispatched() != 0 {
		t.Fatal("an unsigned event must never be processed")
	}
	if entries, _ := ReadEventLog(tr.log); len(entries) != 0 {
		t.Fatal("an unsigned event must not be logged")
	}
}

func TestWebhookDuplicateIsAckedButNotProcessed(t *testing.T) {
	tr := newTestReceiver(t)
	body := eventBody("evt_1")

	first := tr.post("/webhook", body, Sign(testSecret, body))
	second := tr.post("/webhook", body, Sign(testSecret, body))

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses %d, %d; a duplicate must still get 200", first.Code, second.Code)
	}
	if tr.dispatched() != 1 {
		t.Fatalf("dispatched %d times, want 1", tr.dispatched())
	}
}

func TestWebhookRejectsBadJSON(t *testing.T) {
	tr := newTestReceiver(t)
	for _, body := range [][]byte{[]byte(`not json`), []byte(`{"event":"x"}`)} {
		if rec := tr.post("/webhook", body, Sign(testSecret, body)); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", body, rec.Code)
		}
	}
}

func TestWebhookOnlyAcceptsPost(t *testing.T) {
	tr := newTestReceiver(t)
	rec := httptest.NewRecorder()
	tr.srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/webhook", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: status %d, want 405", rec.Code)
	}
}

func TestReplayReprocessesEachLoggedEventOnce(t *testing.T) {
	tr := newTestReceiver(t)
	for _, id := range []string{"evt_1", "evt_2", "evt_1"} { // evt_1 delivered twice
		body := eventBody(id)
		tr.post("/webhook", body, Sign(testSecret, body))
	}
	before := tr.dispatched() // 2

	rec := tr.post("/replay", nil, "")
	var resp map[string]int
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != http.StatusOK || resp["replayed"] != 2 || tr.dispatched() != before+2 {
		t.Fatalf("status %d, replayed %d, dispatched %d->%d; want 200, 2, +2",
			rec.Code, resp["replayed"], before, tr.dispatched())
	}

	if rec := tr.post("/replay?event_id=evt_2", nil, ""); rec.Code != http.StatusOK || tr.dispatched() != before+3 {
		t.Fatalf("single replay: status %d, dispatched %d", rec.Code, tr.dispatched())
	}
	if rec := tr.post("/replay?event_id=nope", nil, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id: status %d, want 404", rec.Code)
	}
}

func TestReplayNeedsTokenWhenConfigured(t *testing.T) {
	tr := newTestReceiver(t)
	tr.rc.ReplayToken = "s3cret"

	if rec := tr.post("/replay", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: status %d, want 401", rec.Code)
	}
	if rec := tr.post("/replay", nil, "", "Authorization", "Bearer s3cret"); rec.Code != http.StatusOK {
		t.Fatalf("right token: status %d, want 200", rec.Code)
	}
}

// ---------- tasks 7 and 8: fan-out + DLQ ----------

func TestDispatcherFansOutAndDeadLettersFailingHandler(t *testing.T) {
	dlqPath := filepath.Join(t.TempDir(), "dead_letter.jsonl")
	dlq, err := OpenJSONL(dlqPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dlq.Close()

	var mu sync.Mutex
	calls := map[string]int{}
	handler := func(name string, fail bool) Handler {
		return Handler{Name: name, Fn: func(context.Context, Event) error {
			mu.Lock()
			calls[name]++
			mu.Unlock()
			if fail {
				return errors.New("down")
			}
			return nil
		}}
	}

	d := NewDispatcher([]Handler{
		handler("fraud", false),
		handler("email", true), // always fails
		handler("analytics", false),
	}, dlq, DispatcherConfig{MaxAttempts: 3, RetryDelay: time.Millisecond})

	for _, id := range []string{"evt_1", "evt_2"} {
		if err := d.Dispatch(Event{Event: "payment.completed", EventID: id}); err != nil {
			t.Fatal(err)
		}
	}
	d.Close() // waits until every handler is done

	if calls["fraud"] != 2 || calls["analytics"] != 2 {
		t.Fatalf("calls = %v: every healthy handler must get every event once", calls)
	}
	if calls["email"] != 6 {
		t.Fatalf("email called %d times, want 2 events x 3 attempts = 6", calls["email"])
	}

	raw, _ := os.ReadFile(dlqPath)
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("dead_letter.jsonl has %d lines, want 2:\n%s", len(lines), raw)
	}
	var dl DeadLetter
	if err := json.Unmarshal(lines[0], &dl); err != nil {
		t.Fatal(err)
	}
	if dl.Handler != "email" || dl.Attempts != 3 || dl.Error != "down" || dl.Event.EventID == "" {
		t.Fatalf("dead letter = %+v", dl)
	}

	if err := d.Dispatch(Event{EventID: "late"}); !errors.Is(err, ErrDispatcherClosed) {
		t.Fatalf("Dispatch after Close: %v, want ErrDispatcherClosed", err)
	}
}

func TestDispatcherSurvivesPanickingHandler(t *testing.T) {
	dlq, err := OpenJSONL(filepath.Join(t.TempDir(), "dlq.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer dlq.Close()

	d := NewDispatcher([]Handler{{Name: "buggy", Fn: func(context.Context, Event) error {
		panic("nil pointer")
	}}}, dlq, DispatcherConfig{MaxAttempts: 2})
	_ = d.Dispatch(Event{EventID: "evt_1"})
	d.Close() // must not crash the test process
}

// ---------- task 4: sender with exponential backoff ----------

func TestSenderRetriesUntilSuccess(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) // fails twice
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &Sender{URL: srv.URL, Secret: testSecret, BaseDelay: time.Millisecond}
	attempts, err := s.Send(context.Background(), Event{Event: "x", EventID: "evt_1"})
	if err != nil || attempts != 3 {
		t.Fatalf("attempts %d, err %v; want 3, nil", attempts, err)
	}
}

func TestSenderGivesUpAfterMaxAttempts(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := &Sender{URL: srv.URL, Secret: testSecret, MaxAttempts: 5, BaseDelay: time.Millisecond}
	attempts, err := s.Send(context.Background(), Event{Event: "x", EventID: "evt_1"})
	if err == nil || attempts != 5 || hits.Load() != 5 {
		t.Fatalf("attempts %d, hits %d, err %v; want 5, 5, error", attempts, hits.Load(), err)
	}
}

func TestSenderDoesNotRetryClientErrors(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnauthorized) // e.g. bad signature
	}))
	defer srv.Close()

	s := &Sender{URL: srv.URL, Secret: testSecret, BaseDelay: time.Millisecond}
	_, err := s.Send(context.Background(), Event{Event: "x", EventID: "evt_1"})
	if !errors.Is(err, ErrPermanent) || hits.Load() != 1 {
		t.Fatalf("hits %d, err %v; want 1 attempt and ErrPermanent", hits.Load(), err)
	}
}

func TestSenderSignsBody(t *testing.T) {
	var verified atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r.Body)
		verified.Store(Verify(testSecret, buf.Bytes(), r.Header.Get(SignatureHeader)))
	}))
	defer srv.Close()

	s := &Sender{URL: srv.URL, Secret: testSecret}
	if _, err := s.Send(context.Background(), Event{Event: "x", EventID: "evt_1"}); err != nil || !verified.Load() {
		t.Fatalf("err %v, signature valid %v", err, verified.Load())
	}
}

func TestBackoffDoublesWithJitterAndCap(t *testing.T) {
	s := &Sender{BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}
	for attempt, full := range map[int]time.Duration{
		1: 100 * time.Millisecond,
		2: 200 * time.Millisecond,
		3: 400 * time.Millisecond,
		4: 800 * time.Millisecond,
		5: time.Second, // 1.6s capped
	} {
		for range 20 {
			got := s.backoff(attempt)
			if got < full/2 || got > full {
				t.Fatalf("attempt %d: wait %s, want between %s and %s", attempt, got, full/2, full)
			}
		}
	}
}
