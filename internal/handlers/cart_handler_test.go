package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/services"
	"checkout-api/internal/validation"
)

func authorizedRequest(method, path string, body *strings.Reader) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, body)
	}
	req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
	return req
}

func TestHandler_UpsertCartItem(t *testing.T) {
	internalErr := errors.New("db write failed")
	tests := []struct {
		name       string
		path       string
		body       string
		authorized bool
		svc        *fakeCartService
		wantStatus int
	}{
		{name: "ok: item added", path: "/user/cart/items/5", body: `{"quantity":2}`, authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusNoContent},
		{name: "unauthorized: token missing", path: "/user/cart/items/5", body: `{"quantity":2}`, svc: &fakeCartService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: invalid item id", path: "/user/cart/items/abc", body: `{"quantity":2}`, authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusBadRequest},
		{name: "bad request: malformed json", path: "/user/cart/items/5", body: `{not-json`, authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusBadRequest},
		{name: "validation: quantity must be positive", path: "/user/cart/items/5", body: `{"quantity":0}`, authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusUnprocessableEntity},
		{name: "not found: item missing", path: "/user/cart/items/5", body: `{"quantity":2}`, authorized: true, svc: &fakeCartService{upsertErr: services.ErrItemNotFound}, wantStatus: http.StatusNotFound},
		{name: "validation: insufficient stock", path: "/user/cart/items/5", body: `{"quantity":2}`, authorized: true, svc: &fakeCartService{upsertErr: validation.ErrInsufficientStock}, wantStatus: http.StatusUnprocessableEntity},
		{name: "unknown service error maps to 500", path: "/user/cart/items/5", body: `{"quantity":2}`, authorized: true, svc: &fakeCartService{upsertErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, tt.svc, nil)
			req := httptest.NewRequest(http.MethodPatch, tt.path, strings.NewReader(tt.body))
			if tt.authorized {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusNoContent {
				if !tt.svc.upsertCalled {
					t.Fatal("expected cart service UpsertCartItem to be called")
				}
				if tt.svc.userID != 7 || tt.svc.itemID != 5 || tt.svc.quantity != 2 {
					t.Fatalf("service args = (%d,%d,%d), want (7,5,2)", tt.svc.userID, tt.svc.itemID, tt.svc.quantity)
				}
			}
		})
	}
}

func TestHandler_RemoveCartItem(t *testing.T) {
	internalErr := errors.New("delete failed")
	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeCartService
		wantStatus int
	}{
		{name: "ok: item removed", path: "/user/cart/items/5", authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusNoContent},
		{name: "unauthorized", path: "/user/cart/items/5", svc: &fakeCartService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: invalid id", path: "/user/cart/items/abc", authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusBadRequest},
		{name: "unknown service error maps to 500", path: "/user/cart/items/5", authorized: true, svc: &fakeCartService{removeErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, tt.svc, nil)
			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			if tt.authorized {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandler_GetUserCart(t *testing.T) {
	internalErr := errors.New("cart read failed")
	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeCartService
		wantStatus int
	}{
		{name: "ok: returns cart", path: "/user/cart?limit=10&offset=0", authorized: true, svc: &fakeCartService{offsetResult: services.CartOffsetResult{Items: []models.CartItem{{ID: 1, Quantity: 2}}, Limit: 10, Offset: 0, Total: 1000}}, wantStatus: http.StatusOK},
		{name: "unauthorized", path: "/user/cart", svc: &fakeCartService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: invalid limit", path: "/user/cart?limit=abc", authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusBadRequest},
		{name: "bad request: negative offset", path: "/user/cart?offset=-1", authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusBadRequest},
		{name: "unknown service error maps to 500", path: "/user/cart", authorized: true, svc: &fakeCartService{offsetErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, tt.svc, nil)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authorized {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandler_GetUserCartCursor(t *testing.T) {
	internalErr := errors.New("cursor read failed")
	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeCartService
		wantStatus int
	}{
		{name: "ok: returns cursor page", path: "/user/cart/cursor?limit=10", authorized: true, svc: &fakeCartService{cursorResult: services.CartCursorResult{Items: []models.CartItem{{ID: 1}}, HasMore: false, Total: 1}}, wantStatus: http.StatusOK},
		{name: "unauthorized", path: "/user/cart/cursor", svc: &fakeCartService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: invalid limit", path: "/user/cart/cursor?limit=abc", authorized: true, svc: &fakeCartService{}, wantStatus: http.StatusBadRequest},
		{name: "bad request: invalid cursor", path: "/user/cart/cursor?cursor=bad", authorized: true, svc: &fakeCartService{cursorErr: services.ErrInvalidCursor}, wantStatus: http.StatusBadRequest},
		{name: "unknown service error maps to 500", path: "/user/cart/cursor", authorized: true, svc: &fakeCartService{cursorErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, tt.svc, nil)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authorized {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
