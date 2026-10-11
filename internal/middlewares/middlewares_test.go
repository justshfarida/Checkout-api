package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("middleware-test-secret")

func signToken(t *testing.T, method jwt.SigningMethod, key any, subject string, expiresIn time.Duration) string {
	t.Helper()

	token := jwt.NewWithClaims(method, jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
	})
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// protected returns a handler that records the user ID it received.
func protected(gotUserID *int, called *bool) http.Handler {
	return AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		*gotUserID, _ = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}), testSecret)
}

func TestAuthMiddleware(t *testing.T) {
	valid := signToken(t, jwt.SigningMethodHS256, testSecret, "42", time.Hour)

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantUserID int
	}{
		{name: "valid bearer token passes user ID to handler", header: "Bearer " + valid, wantStatus: http.StatusOK, wantUserID: 42},
		{name: "scheme is case-insensitive", header: "bearer " + valid, wantStatus: http.StatusOK, wantUserID: 42},
		{name: "missing header", header: "", wantStatus: http.StatusUnauthorized},
		{name: "non-bearer scheme", header: "Basic dXNlcjpwYXNz", wantStatus: http.StatusUnauthorized},
		{name: "bearer without token", header: "Bearer   ", wantStatus: http.StatusUnauthorized},
		{name: "expired token", header: "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, "42", -time.Minute), wantStatus: http.StatusUnauthorized},
		{name: "token signed with another secret", header: "Bearer " + signToken(t, jwt.SigningMethodHS256, []byte("other"), "42", time.Hour), wantStatus: http.StatusUnauthorized},
		{name: "unsigned (alg none) token", header: "Bearer " + signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, "42", time.Hour), wantStatus: http.StatusUnauthorized},
		{name: "non-numeric subject", header: "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, "admin", time.Hour), wantStatus: http.StatusUnauthorized},
		{name: "zero user ID", header: "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, strconv.Itoa(0), time.Hour), wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int
			var called bool

			req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			protected(&gotUserID, &called).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if called != (tt.wantStatus == http.StatusOK) {
				t.Fatalf("protected handler called = %v", called)
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("user ID in context = %d, want %d", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestUserIDFromContext_MissingReturnsFalse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if _, ok := UserIDFromContext(req.Context()); ok {
		t.Error("UserIDFromContext() ok = true for a request without authentication")
	}
}

func TestCORS(t *testing.T) {
	const frontend = "http://checkout.local"

	t.Run("preflight is answered without reaching the handler", func(t *testing.T) {
		called := false
		h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }), frontend)

		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/orders", nil))

		if w.Code != http.StatusNoContent || called {
			t.Fatalf("status = %d, handler called = %v; want 204 and not called", w.Code, called)
		}
	})

	t.Run("allows the frontend origin with credentials and the headers the API needs", func(t *testing.T) {
		h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), frontend)

		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/items", nil))

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != frontend {
			t.Errorf("Allow-Origin = %q, want %q", got, frontend)
		}
		if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("Allow-Credentials = %q, want true (refresh token cookie)", got)
		}
		allowed := w.Header().Get("Access-Control-Allow-Headers")
		for _, h := range []string{"Authorization", "Idempotency-Key", "Content-Type"} {
			if !containsToken(allowed, h) {
				t.Errorf("Allow-Headers %q is missing %s", allowed, h)
			}
		}
	})
}

func containsToken(list, want string) bool {
	for _, part := range strings.Split(list, ",") {
		if strings.TrimSpace(part) == want {
			return true
		}
	}
	return false
}
