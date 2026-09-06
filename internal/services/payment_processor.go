package services

import (
	"fmt"
	"time"
)

type PaymentProcessor interface {
	ProcessPayment(
		amount int,
	) (PaymentResult, error)
}

type MockPaymentProcessor struct{}

func (MockPaymentProcessor) ProcessPayment(
	amount int,
) (PaymentResult, error) {
	amount /= 100

	if amount > 0 && amount < 40 {
		return PaymentResult{
			Success: true,
			TransactionID: fmt.Sprintf(
				"txn_%d",
				time.Now().UnixNano(),
			),
		}, nil
	}

	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}, nil
}
