package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checkout-api/internal/models"
)

// tenMegabyteJSON is a syntactically valid start of a JSON body that is far
// over the 1 MB limit.
func tenMegabyteJSON() string {
	return `{"email":"` + strings.Repeat("a", 10<<20) + `","password":"password123"}`
}

func TestBodyOverOneMegabyteIsRejectedWith413(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		auth   bool
	}{
		{name: "login", method: http.MethodPost, path: "/login"},
		{name: "signup", method: http.MethodPost, path: "/signup"},
		{name: "cart", method: http.MethodPatch, path: "/user/cart/items/1", auth: true},
		{name: "webhook", method: http.MethodPost, path: "/webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandlerWithXsolla(&fakeAuthService{}, nil, &fakeCartService{}, &fakeOrderService{}, &fakeTokenService{}, &fakeXsollaService{signatureValid: true})

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tenMegabyteJSON()))
			if tt.auth {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(1))
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413; body=%.200s", w.Code, w.Body.String())
			}
			if got := errorCode(t, w); got != "PAYLOAD_TOO_LARGE" {
				t.Errorf("error code = %q, want PAYLOAD_TOO_LARGE", got)
			}
		})
	}
}

func TestLogin_UnknownFieldIsRejected(t *testing.T) {
	h := newTestHandler(&fakeAuthService{}, nil, nil, nil, &fakeTokenService{})

	req := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"a@b.com","password":"password123","is_admin":true}`))
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an unknown field", w.Code)
	}
}

func loginRequest(remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"a@b.com","password":"password123"}`))
	req.RemoteAddr = remoteAddr
	return req
}

func TestLogin_EleventhAttemptInAMinuteGets429(t *testing.T) {
	auth := &fakeAuthService{authenticateUser: models.User{ID: 1}}
	token := &fakeTokenService{accessToken: "a", refreshToken: "r", refreshExpiresAt: time.Now().Add(time.Hour)}
	h := newTestHandler(auth, nil, nil, nil, token)
	routes := h.Routes()

	for i := 1; i <= 10; i++ {
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, loginRequest("203.0.113.7:5000"))
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d got 429; the first 10 must be allowed", i)
		}
	}

	w := httptest.NewRecorder()
	routes.ServeHTTP(w, loginRequest("203.0.113.7:5001")) // same IP, new port
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("11th attempt status = %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 response must include Retry-After")
	}

	// Another client is not affected by the first one's limit.
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, loginRequest("198.51.100.9:5000"))
	if w.Code == http.StatusTooManyRequests {
		t.Error("a different IP was rate limited")
	}
}
