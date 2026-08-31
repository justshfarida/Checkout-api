package service

import (
	"checkout-api/internal/controllers/dtos"
	"math"
	"time"
)

type PaymentService struct{}

func NewPaymentService() *PaymentService {
	return &PaymentService{}
}

func (s *PaymentService) ProcessPayment(amount int) dtos.PaymentResult {
	var result dtos.PaymentResult
	retryWithBackoff(3, func() (bool, error) {
		result = mockProcessPayment(amount)
		return result.Success, nil
	})
	return result
}

func retryWithBackoff(maxRetries int, fn func() (bool, error)) bool {
	baseDelay := 100 * time.Millisecond
	maxDelay := 400 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		ok, err := fn()
		if err == nil && ok {
			return true
		}
		if attempt < maxRetries-1 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
			if delay > maxDelay {
				delay = maxDelay
			}
			time.Sleep(delay)
		}
	}
	return false
}

func mockProcessPayment(amount int) dtos.PaymentResult {
	if amount > 0 && amount < 1000000 {
		return dtos.PaymentResult{Success: true}
	}
	return dtos.PaymentResult{Success: false, Error: "Payment declined"}
}
