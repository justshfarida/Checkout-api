package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/services"
)

func TestHandler_CreateOrder(t *testing.T) {
	internalErr := errors.New("order repository failed")
	tests := []struct {
		name           string
		authorized     bool
		idempotencyKey string
		svc            *fakeOrderService
		wantStatus     int
	}{
		{
			name:           "ok: paid order created",
			authorized:     true,
			idempotencyKey: "key-paid",
			svc: &fakeOrderService{createResult: &services.CreateOrderResult{
				Order:   &models.Order{ID: 1, UserID: 7, Total: 1500, Status: "paid"},
				Payment: services.PaymentResult{Success: true, TransactionID: "tx-1"},
			}},
			wantStatus: http.StatusCreated,
		},
		{
			name:           "payment declined maps to 402",
			authorized:     true,
			idempotencyKey: "key-declined",
			svc: &fakeOrderService{createResult: &services.CreateOrderResult{
				Order:   &models.Order{ID: 2, UserID: 7, Total: 5000, Status: "failed"},
				Payment: services.PaymentResult{Success: false, Error: "Payment declined"},
			}},
			wantStatus: http.StatusPaymentRequired,
		},
		{name: "unauthorized", svc: &fakeOrderService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: idempotency key missing", authorized: true, svc: &fakeOrderService{}, wantStatus: http.StatusBadRequest},
		{name: "unknown service error maps to 500", authorized: true, idempotencyKey: "key-error", svc: &fakeOrderService{createErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, nil, tt.svc)
			req := httptest.NewRequest(http.MethodPost, "/orders", nil)
			if tt.authorized {
				req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
			}
			if tt.idempotencyKey != "" {
				req.Header.Set("Idempotency-Key", tt.idempotencyKey)
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandler_CreateOrder_IdempotencyPreventsDuplicateServiceCall(t *testing.T) {
	svc := &fakeOrderService{createResult: &services.CreateOrderResult{
		Order:   &models.Order{ID: 1, UserID: 7, Total: 1500, Status: "paid"},
		Payment: services.PaymentResult{Success: true, TransactionID: "tx-1"},
	}}
	h := newTestHandler(nil, nil, nil, svc)
	router := h.Routes()

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/orders", nil)
		req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
		req.Header.Set("Idempotency-Key", "same-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d", i+1, w.Code, http.StatusCreated)
		}
	}

	if svc.createCalls != 1 {
		t.Fatalf("CreateOrder called %d times, want 1", svc.createCalls)
	}
}

func TestHandler_GetUserOrders(t *testing.T) {
	internalErr := errors.New("orders read failed")
	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeOrderService
		wantStatus int
	}{
		{name: "ok: returns orders", path: "/user/orders?limit=10&offset=0", authorized: true, svc: &fakeOrderService{offsetResult: services.OrderOffsetResult{Orders: []models.UserOrder{{ID: 1, UserID: 7, Total: 1000, Status: "paid"}}, Limit: 10, Offset: 0}}, wantStatus: http.StatusOK},
		{name: "unauthorized", path: "/user/orders", svc: &fakeOrderService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: invalid limit", path: "/user/orders?limit=abc", authorized: true, svc: &fakeOrderService{}, wantStatus: http.StatusBadRequest},
		{name: "bad request: negative offset", path: "/user/orders?offset=-1", authorized: true, svc: &fakeOrderService{}, wantStatus: http.StatusBadRequest},
		{name: "unknown service error maps to 500", path: "/user/orders", authorized: true, svc: &fakeOrderService{offsetErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, nil, tt.svc)
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

func TestHandler_GetUserOrdersCursor(t *testing.T) {
	internalErr := errors.New("orders cursor failed")
	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeOrderService
		wantStatus int
	}{
		{name: "ok: returns cursor page", path: "/user/orders/cursor?limit=10", authorized: true, svc: &fakeOrderService{cursorResult: services.OrderCursorResult{Orders: []models.UserOrder{{ID: 1}}, HasMore: false}}, wantStatus: http.StatusOK},
		{name: "unauthorized", path: "/user/orders/cursor", svc: &fakeOrderService{}, wantStatus: http.StatusUnauthorized},
		{name: "bad request: invalid limit", path: "/user/orders/cursor?limit=abc", authorized: true, svc: &fakeOrderService{}, wantStatus: http.StatusBadRequest},
		{name: "bad request: invalid cursor", path: "/user/orders/cursor?cursor=bad", authorized: true, svc: &fakeOrderService{cursorErr: services.ErrInvalidCursor}, wantStatus: http.StatusBadRequest},
		{name: "unknown service error maps to 500", path: "/user/orders/cursor", authorized: true, svc: &fakeOrderService{cursorErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, nil, tt.svc)
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
