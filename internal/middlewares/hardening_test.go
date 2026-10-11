package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/items", nil))

	want := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
		"Referrer-Policy":           "no-referrer",
		"Cache-Control":             "no-store",
	}
	for name, value := range want {
		if got := w.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestSecurityHeaders_SwaggerKeepsWorking(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))

	if got := w.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("swagger page got CSP %q; its scripts would be blocked", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("swagger page still needs nosniff, got %q", got)
	}
}

func TestRateLimiter_WindowResets(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	l := NewRateLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	if ok, _ := l.allow("ip"); !ok {
		t.Fatal("1st request blocked")
	}
	if ok, _ := l.allow("ip"); !ok {
		t.Fatal("2nd request blocked")
	}
	ok, retry := l.allow("ip")
	if ok {
		t.Fatal("3rd request allowed, limit is 2")
	}
	if retry != time.Minute {
		t.Errorf("retry after = %v, want 1m", retry)
	}

	now = now.Add(time.Minute)
	if ok, _ := l.allow("ip"); !ok {
		t.Error("request after the window was still blocked")
	}
}

func TestRateLimiter_ForgetsIdleClients(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	l := NewRateLimiter(10, time.Minute)
	l.now = func() time.Time { return now }

	l.allow("a")
	l.allow("b")
	now = now.Add(2 * time.Minute)
	l.allow("c")

	if len(l.clients) != 1 {
		t.Errorf("clients kept = %d, want 1 (idle ones swept)", len(l.clients))
	}
}

func TestRateLimiter_Returns429WithRetryAfter(t *testing.T) {
	l := NewRateLimiter(1, time.Minute)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/login", nil))
	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/login", nil))

	if first.Code != http.StatusOK || second.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = %d, %d; want 200, 429", first.Code, second.Code)
	}
	if second.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want 60", second.Header().Get("Retry-After"))
	}
}
