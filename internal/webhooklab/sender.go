package webhooklab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"time"
)

// ErrPermanent means retrying cannot help (e.g. 400 or 401): the sender
// stops at once instead of using up its attempts.
var ErrPermanent = errors.New("permanent failure")

// Sender delivers events like a webhook provider does (task 4): sign the
// body, POST it, and retry with exponential backoff until a 2xx arrives or
// MaxAttempts is used up.
type Sender struct {
	URL         string
	Secret      string
	Client      *http.Client  // nil = 10s timeout client
	MaxAttempts int           // default 5
	BaseDelay   time.Duration // wait after the 1st failure (default 500ms)
	MaxDelay    time.Duration // cap for one wait (default 30s)
}

// Send delivers one event and returns how many attempts it took.
func (s *Sender) Send(ctx context.Context, e Event) (int, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return 0, fmt.Errorf("encode event: %w", err)
	}
	return s.SendRaw(ctx, body, Sign(s.Secret, body))
}

// SendRaw sends an already encoded body with the given signature (lets the
// demo send a wrongly signed request).
func (s *Sender) SendRaw(ctx context.Context, body []byte, signature string) (int, error) {
	maxAttempts := s.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		status, err := s.post(ctx, body, signature)
		switch {
		case err == nil && status >= 200 && status < 300:
			return attempt, nil
		case err == nil && status >= 400 && status < 500 && status != http.StatusTooManyRequests:
			// Our request is wrong (bad signature, bad JSON): the same
			// request will fail the same way, so retrying is pointless.
			return attempt, fmt.Errorf("%w: HTTP %d", ErrPermanent, status)
		case err == nil:
			lastErr = fmt.Errorf("HTTP %d", status) // 5xx or 429: try again
		default:
			lastErr = err // receiver down, timeout, connection refused
		}

		if attempt == maxAttempts {
			break
		}
		wait := s.backoff(attempt)
		log.Printf("attempt %d/%d failed (%v), retrying in %s", attempt, maxAttempts, lastErr, wait.Round(time.Millisecond))
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return attempt, ctx.Err()
		}
	}
	return maxAttempts, fmt.Errorf("gave up after %d attempts: %w", maxAttempts, lastErr)
}

// backoff doubles the wait each attempt (base, 2x, 4x, 8x ...), capped at
// MaxDelay, with random jitter: the wait is between 50% and 100% of that.
// Jitter spreads retries out, so many senders that failed at the same
// moment do not all hit the receiver again at the same moment.
func (s *Sender) backoff(attempt int) time.Duration {
	base := s.BaseDelay
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	limit := s.MaxDelay
	if limit <= 0 {
		limit = 30 * time.Second
	}

	d := base << (attempt - 1)
	if d > limit || d <= 0 { // d <= 0: the shift overflowed
		d = limit
	}
	half := d / 2
	return half + rand.N(half+1) // #nosec G404 -- jitter only, not security
}

func (s *Sender) post(ctx context.Context, body []byte, signature string) (int, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(SignatureHeader, signature)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // read it fully so the connection is reused
	return resp.StatusCode, nil
}
