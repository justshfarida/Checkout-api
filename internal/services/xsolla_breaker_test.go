package services

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/resilience"
)

func TestCreatePayment_BreakerOpensAndFailsFastWhenXsollaIsDown(t *testing.T) {
	service, calls := scriptedXsolla(t, 503) // Xsolla keeps failing

	for i := 0; i < paymentBreakerThreshold; i++ {
		if _, err := service.CreatePayment(context.Background(), 1, i, nil); err == nil {
			t.Fatalf("call %d: error = nil, want failure", i+1)
		}
	}

	if got := service.breaker.State(); got != resilience.Open {
		t.Fatalf("breaker state = %v after %d failed payments, want open", got, paymentBreakerThreshold)
	}

	callsBefore := *calls
	_, err := service.CreatePayment(context.Background(), 1, 99, nil)

	if !errors.Is(err, ErrPaymentProviderUnavailable) {
		t.Fatalf("error = %v, want ErrPaymentProviderUnavailable while open", err)
	}
	if *calls != callsBefore {
		t.Errorf("Xsolla was called %d more times; an open breaker must fail fast", *calls-callsBefore)
	}
}

func TestCreatePayment_ClientErrorsDoNotTripTheBreaker(t *testing.T) {
	service, _ := scriptedXsolla(t, 422) // Xsolla answers, the request is bad

	for i := 0; i < paymentBreakerThreshold+2; i++ {
		_, _ = service.CreatePayment(context.Background(), 1, i, nil)
	}

	if got := service.breaker.State(); got != resilience.Closed {
		t.Fatalf("breaker state = %v, want closed: 4xx means Xsolla is healthy", got)
	}
}

func TestCreatePayment_SuccessKeepsBreakerClosed(t *testing.T) {
	service, _ := scriptedXsolla(t, 503, 503, 200) // recovers within the retries

	if _, err := service.CreatePayment(context.Background(), 1, 1, nil); err != nil {
		t.Fatalf("error = %v, want success", err)
	}
	if got := service.breaker.State(); got != resilience.Closed {
		t.Fatalf("breaker state = %v, want closed", got)
	}
}
