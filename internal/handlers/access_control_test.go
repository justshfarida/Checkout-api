package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"checkout-api/internal/models"
)

// Broken access control (OWASP A01): a logged-in user must never read
// another user's order by changing the ID in the URL. "Not yours" answers
// 404 like "doesn't exist", so nobody can find out which order IDs exist.
func TestGetUserOrder_AccessControl(t *testing.T) {
	const orderOwner = 7
	orders := &fakeOrderService{ordersByID: map[int]models.UserOrder{
		42: {ID: 42, UserID: orderOwner, Total: 500, Status: "paid"},
	}}

	tests := []struct {
		name       string
		userID     int // 0 = no token
		wantStatus int
	}{
		{name: "owner", userID: orderOwner, wantStatus: http.StatusOK},
		{name: "other user", userID: 8, wantStatus: http.StatusNotFound},
		{name: "anonymous", userID: 0, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(&fakeAuthService{}, nil, nil, orders, nil)

			req := httptest.NewRequest(http.MethodGet, "/user/orders/42", nil)
			if tt.userID != 0 {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(tt.userID))
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var got models.UserOrder
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.ID != 42 {
					t.Errorf("body = %s, want order 42", w.Body.String())
				}
			}
		})
	}
}

func TestGetUserOrder_MissingAndForeignOrderLookIdentical(t *testing.T) {
	orders := &fakeOrderService{ordersByID: map[int]models.UserOrder{
		42: {ID: 42, UserID: 7},
	}}
	h := newTestHandler(&fakeAuthService{}, nil, nil, orders, nil)

	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+testAccessToken(8))
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, req)
		return w
	}

	foreign := get("/user/orders/42")  // exists, belongs to user 7
	missing := get("/user/orders/999") // doesn't exist

	if foreign.Code != missing.Code || foreign.Body.String() != missing.Body.String() {
		t.Errorf("foreign = %d %s, missing = %d %s; they must be identical",
			foreign.Code, foreign.Body.String(), missing.Code, missing.Body.String())
	}
}

func TestGetUserOrder_InvalidIDIs404(t *testing.T) {
	h := newTestHandler(&fakeAuthService{}, nil, nil, &fakeOrderService{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/user/orders/abc", nil)
	req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
