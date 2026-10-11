package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"checkout-api/internal/services"
)

// gatedOrderService blocks inside CreateOrder until released, simulating a
// slow order creation (DB + Xsolla call), so tests can observe concurrency.
type gatedOrderService struct {
	fakeOrderService

	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
}

func newGatedOrderService() *gatedOrderService {
	return &gatedOrderService{
		fakeOrderService: fakeOrderService{createResult: newCreateOrderResult()},
		started:          make(chan struct{}, 10),
		release:          make(chan struct{}),
	}
}

func (g *gatedOrderService) CreateOrder(
	ctx context.Context,
	userID int,
) (*services.CreateOrderResult, error) {
	g.mu.Lock()
	g.calls++
	g.mu.Unlock()

	g.started <- struct{}{}
	<-g.release

	return g.createResult, nil
}

func (g *gatedOrderService) callCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

func waitStarted(t *testing.T, g *gatedOrderService, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-g.started:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of %d order creations started; requests are being serialized", i, n)
		}
	}
}

func TestCreateOrder_DifferentKeysRunInParallel(t *testing.T) {
	svc := newGatedOrderService()
	router := newTestHandler(nil, nil, nil, svc, nil).Routes()

	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i, key := range []string{"key-a", "key-b"} {
		wg.Add(1)
		go func(i int, key string) {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, createOrderRequest(7, key))
			codes[i] = w.Code
		}(i, key)
	}

	// Both must be inside CreateOrder at the same time. With a global lock
	// held for the whole request, the second would never start.
	waitStarted(t, svc, 2)

	close(svc.release)
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusCreated {
			t.Errorf("request %d status = %d, want 201", i, code)
		}
	}
}

func TestCreateOrder_SameKeyInFlightReturns409(t *testing.T) {
	svc := newGatedOrderService()
	router := newTestHandler(nil, nil, nil, svc, nil).Routes()

	first := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(first, createOrderRequest(7, "dup-key"))
		close(done)
	}()
	waitStarted(t, svc, 1)

	// A duplicate while the first is still running is rejected, not queued.
	dup := httptest.NewRecorder()
	router.ServeHTTP(dup, createOrderRequest(7, "dup-key"))
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate in flight: status = %d, want 409", dup.Code)
	}

	close(svc.release)
	<-done
	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want 201", first.Code)
	}

	// After it finished, the same key replays the stored response.
	replay := httptest.NewRecorder()
	router.ServeHTTP(replay, createOrderRequest(7, "dup-key"))
	if replay.Code != http.StatusCreated || replay.Body.String() != first.Body.String() {
		t.Errorf("replay = %d %s, want the original 201 response", replay.Code, replay.Body.String())
	}

	if got := svc.callCount(); got != 1 {
		t.Errorf("CreateOrder called %d times, want exactly 1 for one key", got)
	}
}
