package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// scriptedXsolla answers each attempt with the next status code in the list.
// A status of 0 simulates a network error.
func scriptedXsolla(t *testing.T, statuses ...int) (*XsollaService, *int32) {
	t.Helper()

	var calls int32
	service := NewXsollaService("12345", "merchant", "key", "wh", "")
	service.retryBase = time.Millisecond // keep tests fast

	service.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		status := statuses[len(statuses)-1]
		if int(n) <= len(statuses) {
			status = statuses[n-1]
		}
		if status == 0 {
			return nil, errors.New("connection reset by peer")
		}
		if status == http.StatusOK {
			return jsonResponse(http.StatusOK, `{"token":"tok","order_id":1}`), nil
		}
		return jsonResponse(status, `{"errorMessage":"failure"}`), nil
	})}

	return service, &calls
}

func TestCreatePayment_RetriesTransientFailures(t *testing.T) {
	tests := []struct {
		name      string
		statuses  []int
		wantCalls int32
	}{
		{name: "two 503s then success", statuses: []int{503, 503, 200}, wantCalls: 3},
		{name: "429 rate limit then success", statuses: []int{429, 200}, wantCalls: 2},
		{name: "network error then success", statuses: []int{0, 200}, wantCalls: 2},
		{name: "first try succeeds", statuses: []int{200}, wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, calls := scriptedXsolla(t, tt.statuses...)

			got, err := service.CreatePayment(context.Background(), 1, 1, nil)

			if err != nil {
				t.Fatalf("CreatePayment() error = %v, want success", err)
			}
			if got.Token != "tok" {
				t.Errorf("token = %q, want tok", got.Token)
			}
			if *calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", *calls, tt.wantCalls)
			}
		})
	}
}

func TestCreatePayment_DoesNotRetryClientErrors(t *testing.T) {
	for _, status := range []int{400, 401, 422} {
		service, calls := scriptedXsolla(t, status)

		_, err := service.CreatePayment(context.Background(), 1, 1, nil)

		if err == nil {
			t.Fatalf("status %d: error = nil, want error", status)
		}
		if *calls != 1 {
			t.Errorf("status %d: calls = %d, want 1 (4xx must not be retried)", status, *calls)
		}
	}
}

func TestCreatePayment_GivesUpAfterMaxAttempts(t *testing.T) {
	service, calls := scriptedXsolla(t, 500)

	_, err := service.CreatePayment(context.Background(), 1, 1, nil)

	if err == nil {
		t.Fatal("error = nil, want error after all attempts fail")
	}
	if *calls != maxPaymentAttempts {
		t.Errorf("calls = %d, want %d", *calls, maxPaymentAttempts)
	}
	if !strings.Contains(err.Error(), "after 3 attempts") || !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("error = %q, want it to mention the attempts and the last status", err)
	}
}

func TestCreatePayment_StopsRetryingWhenContextIsCancelled(t *testing.T) {
	service, calls := scriptedXsolla(t, 503)
	service.retryBase = time.Hour // backoff would be long; cancellation must cut it short

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := service.CreatePayment(ctx, 1, 1, nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("took %v; must stop waiting as soon as the context is cancelled", time.Since(start))
	}
	if *calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry after cancellation)", *calls)
	}
}

func TestNewXsollaService_SetsPerAttemptTimeout(t *testing.T) {
	service := NewXsollaService("p", "m", "k", "w", "")

	if service.client.Timeout != paymentAttemptTimeout {
		t.Errorf("client timeout = %v, want %v", service.client.Timeout, paymentAttemptTimeout)
	}
}
