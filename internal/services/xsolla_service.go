package services

import (
	"bytes"
	"context"
	"crypto/sha1" // #nosec G505 -- Xsolla defines the webhook signature as SHA-1(body + secret); we must match it
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"checkout-api/internal/resilience"
)

const (
	// paymentAttemptTimeout limits one HTTP call to Xsolla. Three attempts
	// plus backoff stay under the server's 15s WriteTimeout.
	paymentAttemptTimeout = 4 * time.Second

	// maxPaymentAttempts is the total number of tries (1 call + 2 retries).
	maxPaymentAttempts = 3

	// defaultRetryBaseDelay is the base for exponential backoff with jitter:
	// attempt 0 waits 0–100ms, attempt 1 waits 0–200ms.
	defaultRetryBaseDelay = 100 * time.Millisecond

	// After this many consecutive failed payment calls (each already
	// retried), stop calling Xsolla for paymentBreakerCooldown.
	paymentBreakerThreshold = 5
	paymentBreakerCooldown  = 30 * time.Second
)

type XsollaService struct {
	projectID     string
	merchantID    string
	apiKey        string
	client        *http.Client
	retryBase     time.Duration
	breaker       *resilience.Breaker
	webhookSecret string
	// returnURL is where Pay Station sends the user after payment. It is set
	// per environment so local and minikube each redirect back to themselves.
	returnURL string
}

func NewXsollaService(
	projectID string,
	merchantID string,
	apiKey string,
	webhookSecret string,
	returnURL string,
) *XsollaService {
	return &XsollaService{
		projectID:     projectID,
		merchantID:    merchantID,
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		returnURL:     returnURL,
		client:        &http.Client{Timeout: paymentAttemptTimeout},
		retryBase:     defaultRetryBaseDelay,
		breaker: resilience.NewBreaker(
			"xsolla-payment",
			paymentBreakerThreshold,
			paymentBreakerCooldown,
		),
	}
}

type xsollaPaymentRequest struct {
	Sandbox bool `json:"sandbox"`

	User struct {
		ID struct {
			Value string `json:"value"`
		} `json:"id"`

		Country struct {
			Value       string `json:"value"`
			AllowModify bool   `json:"allow_modify"`
		} `json:"country"`
	} `json:"user"`

	Purchase struct {
		Items []XsollaItem `json:"items"`
	} `json:"purchase"`
	CustomParameters map[string]string `json:"custom_parameters"`

	Settings *xsollaPaymentSettings `json:"settings,omitempty"`
}

type xsollaPaymentSettings struct {
	ReturnURL string `json:"return_url,omitempty"`
}

type XsollaItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type XsollaPaymentResponse struct {
	Token   string `json:"token"`
	OrderID int    `json:"order_id"`
}

// CreatePayment requests a Pay Station token from Xsolla, protected by a
// circuit breaker (outer) and retries with jitter (inner):
//
//	breaker.Allow -> up to 3 attempts with backoff -> breaker.Success/Failure
//
// When Xsolla keeps failing, the breaker opens and calls fail fast with
// ErrPaymentProviderUnavailable instead of each request waiting on retries.
func (s *XsollaService) CreatePayment(
	ctx context.Context,
	userID int,
	orderID int,
	items []XsollaItem,
) (*XsollaPaymentResponse, error) {
	if err := s.breaker.Allow(); err != nil {
		slog.WarnContext(ctx, "Payment request rejected: circuit breaker open",
			"order_id", orderID,
		)
		return nil, fmt.Errorf("%w: %v", ErrPaymentProviderUnavailable, err)
	}

	result, err := s.createPaymentWithRetry(ctx, userID, orderID, items)

	var retryable *retryableError
	switch {
	case err == nil:
		s.breaker.Success()
	case ctx.Err() != nil:
		// The caller gave up; this says nothing about Xsolla's health.
		s.breaker.Cancel()
	case errors.As(err, &retryable):
		// Network errors, timeouts, 429/5xx even after retries: Xsolla is unhealthy.
		s.breaker.Failure()
	default:
		// 4xx or bad input: Xsolla answered, so it is healthy.
		s.breaker.Success()
	}

	return result, err
}

func (s *XsollaService) createPaymentWithRetry(
	ctx context.Context,
	userID int,
	orderID int,
	items []XsollaItem,
) (*XsollaPaymentResponse, error) {

	var body xsollaPaymentRequest

	body.Sandbox = true
	body.User.ID.Value = strconv.Itoa(userID)
	body.User.Country.Value = "US"
	body.User.Country.AllowModify = false
	body.Purchase.Items = items
	body.CustomParameters = map[string]string{
		"order_id": strconv.Itoa(orderID),
	}

	if s.returnURL != "" {
		body.Settings = &xsollaPaymentSettings{ReturnURL: s.returnURL}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var lastErr error

	for attempt := 0; attempt < maxPaymentAttempts; attempt++ {
		result, err := s.sendPaymentRequest(ctx, data)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Permanent failures (4xx, bad JSON) will not succeed on retry.
		var retryable *retryableError
		if !errors.As(err, &retryable) {
			return nil, err
		}

		if attempt == maxPaymentAttempts-1 {
			break
		}

		// Exponential backoff with full jitter: a random wait in
		// [0, base*2^attempt) so many clients don't retry in lockstep.
		backoff := time.Duration(rand.Int63n(int64(s.retryBase << attempt))) // #nosec G404 -- retry jitter only, not a secret; predictability is harmless

		slog.WarnContext(ctx, "Retrying payment request",
			"order_id", orderID,
			"attempt", attempt+1,
			"backoff_ms", backoff.Milliseconds(),
			"error", err.Error(),
		)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			// The client gave up or the request deadline passed: stop retrying.
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf(
		"payment request failed after %d attempts: %w",
		maxPaymentAttempts,
		lastErr,
	)
}

// retryableError marks a failure that may succeed if tried again:
// network errors, timeouts, HTTP 429 and 5xx.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// sendPaymentRequest performs exactly one HTTP call to Xsolla.
func (s *XsollaService) sendPaymentRequest(
	ctx context.Context,
	data []byte,
) (*XsollaPaymentResponse, error) {
	url := fmt.Sprintf(
		"https://store.xsolla.com/api/v3/project/%s/admin/payment/token",
		s.projectID,
	)

	// A new request (and body reader) is needed for every attempt.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.projectID, s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		// If our own context ended, retrying is pointless.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Network error or per-attempt timeout: worth another try.
		return nil, &retryableError{err: err}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &retryableError{err: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf(
			"xsolla returned %s: %s",
			resp.Status,
			string(responseBody),
		)

		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode >= 500 {
			return nil, &retryableError{err: statusErr}
		}

		return nil, statusErr
	}

	var result XsollaPaymentResponse

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *XsollaService) VerifyXsollaSignature(rawBody []byte, authHeader string) bool {
	const prefix = "Signature "

	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}
	provided := strings.TrimSpace(
		strings.TrimPrefix(authHeader, prefix),
	)
	h := sha1.New() // #nosec G401 -- required by Xsolla's signature spec; compared in constant time below
	h.Write(rawBody)
	h.Write([]byte(s.webhookSecret))

	expected := hex.EncodeToString(
		h.Sum(nil),
	) //give me the SHA-1 result as bytes

	return subtle.ConstantTimeCompare(
		[]byte(strings.ToLower(provided)),
		[]byte(strings.ToLower(expected)),
	) == 1
}
