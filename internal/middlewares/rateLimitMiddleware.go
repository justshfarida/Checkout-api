package middlewares

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter allows at most `limit` requests per `window` from one client IP
// (fixed window). It protects /login from password guessing: 10 tries a
// minute is plenty for a human and far too slow for a brute-force script.
//
// State is in memory, so with several replicas each pod counts on its own.
// A shared store (Redis) would be needed for one global limit.
type RateLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	clients   map[string]*windowCounter
	lastSweep time.Time
	now       func() time.Time // replaced in tests
}

type windowCounter struct {
	start time.Time
	count int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]*windowCounter),
		now:     time.Now,
	}
}

// allow records one request for key and reports whether it may proceed.
// When it may not, it also returns how long until the window resets.
func (l *RateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	c, ok := l.clients[key]
	if !ok || now.Sub(c.start) >= l.window {
		l.clients[key] = &windowCounter{start: now, count: 1}
		return true, 0
	}
	if c.count >= l.limit {
		return false, l.window - now.Sub(c.start)
	}
	c.count++
	return true, 0
}

// sweep forgets clients whose window is over, at most once per window, so
// the map can't grow forever with one entry per IP ever seen.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	for key, c := range l.clients {
		if now.Sub(c.start) >= l.window {
			delete(l.clients, key)
		}
	}
	l.lastSweep = now
}

// Middleware answers 429 Too Many Requests with Retry-After once a client
// is over the limit; otherwise it calls next.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := l.allow(clientIP(r))
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"TOO_MANY_REQUESTS","message":"Too many attempts, please try again later"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP is the TCP peer address. X-Forwarded-For is ignored on purpose:
// any client can send it, so trusting it would let an attacker use a fresh
// "IP" on every request and never hit the limit.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
