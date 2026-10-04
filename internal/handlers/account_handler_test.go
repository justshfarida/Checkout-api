package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checkout-api/internal/models"
)

func findCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestHandler_LoginUser_RefreshCookieLastsSevenDays(t *testing.T) {
	// The refresh token is valid for 7 days, and http.Cookie.MaxAge is in
	// seconds, so the cookie should live 7*24*3600 = 604800 seconds.
	auth := &fakeAuthService{authenticateUser: models.User{ID: 1}}
	token := &fakeTokenService{
		accessToken:      "access",
		refreshToken:     "refresh",
		refreshExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	h := newTestHandler(auth, nil, nil, nil, token)

	req := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"a@b.com","password":"password123"}`))
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	c := findCookie(w, "refresh_token")
	if c == nil {
		t.Fatal("refresh_token cookie not set")
	}
	if c.MaxAge != 604800 {
		t.Errorf("refresh cookie MaxAge = %d seconds, want 604800 (7 days)", c.MaxAge)
	}
	if !c.HttpOnly {
		t.Error("refresh cookie must be HttpOnly")
	}
}

func TestHandler_Logout(t *testing.T) {
	tests := []struct {
		name       string
		cookie     *http.Cookie
		logoutErr  error
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "revokes refresh token from cookie",
			cookie:     &http.Cookie{Name: "refresh_token", Value: "rt-1"},
			wantStatus: http.StatusNoContent,
			wantCalled: true,
		},
		{
			name:       "without cookie still succeeds",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "revocation failure returns 500",
			cookie:     &http.Cookie{Name: "refresh_token", Value: "rt-1"},
			logoutErr:  errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &fakeTokenService{logoutErr: tt.logoutErr}
			h := newTestHandler(&fakeAuthService{}, nil, nil, nil, token)

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if token.logoutCalled != tt.wantCalled {
				t.Fatalf("Logout called = %v, want %v", token.logoutCalled, tt.wantCalled)
			}
			if tt.wantCalled && token.logoutToken != "rt-1" {
				t.Errorf("revoked token %q, want rt-1", token.logoutToken)
			}
			if tt.wantStatus == http.StatusNoContent {
				c := findCookie(w, "refresh_token")
				if c == nil || c.MaxAge >= 0 || c.Value != "" {
					t.Errorf("refresh cookie not cleared: %+v", c)
				}
			}
		})
	}
}

func TestHandler_GetCurrentUser(t *testing.T) {
	t.Run("returns profile of the token's user without password hash", func(t *testing.T) {
		auth := &fakeAuthService{userByID: models.User{
			ID: 7, Email: "me@example.com", Balance: 250, Hash: []byte("$2a$10$secret-hash"),
		}}
		h := newTestHandler(auth, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
		req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
		}
		if auth.getUserByIDArg != 7 {
			t.Errorf("looked up user %d, want 7", auth.getUserByIDArg)
		}
		if strings.Contains(w.Body.String(), "secret-hash") {
			t.Fatalf("response leaks the password hash: %s", w.Body.String())
		}

		var body struct {
			ID      int    `json:"id"`
			Email   string `json:"email"`
			Balance int    `json:"balance"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if body.ID != 7 || body.Email != "me@example.com" || body.Balance != 250 {
			t.Errorf("body = %+v, want id 7, email me@example.com, balance 250", body)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		h := newTestHandler(&fakeAuthService{}, nil, nil, nil, nil)

		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/user/me", nil))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

func TestHandler_GetUserInventory(t *testing.T) {
	t.Run("returns the token user's inventory", func(t *testing.T) {
		auth := &fakeAuthService{inventory: []models.InventoryItem{{ID: 3, Name: "Dragon", Quantity: 2}}}
		h := newTestHandler(auth, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/user/inventory", nil)
		req.Header.Set("Authorization", "Bearer "+testAccessToken(9))
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if auth.inventoryUserID != 9 {
			t.Errorf("inventory loaded for user %d, want 9", auth.inventoryUserID)
		}
		var got []models.InventoryItem
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got) != 1 || got[0].Quantity != 2 {
			t.Errorf("inventory = %+v (err %v), want one item with quantity 2", got, err)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		auth := &fakeAuthService{inventoryErr: errors.New("db down")}
		h := newTestHandler(auth, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/user/inventory", nil)
		req.Header.Set("Authorization", "Bearer "+testAccessToken(9))
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})
}
