package service

import (
	"checkout-api/internal/controllers/dtos"
	"encoding/json"
	"math"
	"net/http"
	"time"
)

type Service struct{}

func New() *Service { return &Service{} }

// Retry  with backoff

// BUSINESS

func retryWithBackoff(maxRetries int, fn func() (bool, error)) bool {
	baseDelay := 100 * time.Millisecond
	maxDelay := 400 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		ok, err := fn()
		if err == nil && ok {
			return true
		}

		if attempt < maxRetries {
			delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
			if delay > maxDelay {
				delay = maxDelay
			}
			time.Sleep(delay)
		}
	}
	return false
}

// BUSINESS

func mockProcessPayment(amount int) dtos.PaymentResult {
	if amount > 0 && amount < 1000000 {
		return dtos.PaymentResult{
			Success: true,
		}
	}
	return dtos.PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
