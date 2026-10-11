package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/services"
	"checkout-api/internal/validation"
)

func newCreateOrderResult() *services.CreateOrderResult {
	return &services.CreateOrderResult{
		Order: &models.Order{
			ID:     1,
			UserID: 7,
			Total:  1500,
			Status: "pending",
		},
		XsollaOrderID: 555,
		PaymentURL:    "https://sandbox-secure.xsolla.com/paystation4/?token=tok",
	}
}

func createOrderRequest(userID int, idempotencyKey string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/orders", nil)

	if userID > 0 {
		req.Header.Set("Authorization", "Bearer "+testAccessToken(userID))
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	return req
}

func TestHandler_CreateOrder(t *testing.T) {
	internalErr := errors.New(
		"order repository failed",
	)

	tests := []struct {
		name           string
		userID         int
		idempotencyKey string
		svc            *fakeOrderService
		wantStatus     int
		wantCode       string
	}{
		{
			name:           "ok: pending order created with payment link",
			userID:         7,
			idempotencyKey: "key-ok",
			svc:            &fakeOrderService{createResult: newCreateOrderResult()},
			wantStatus:     http.StatusCreated,
		},
		{
			name:       "unauthorized",
			svc:        &fakeOrderService{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "bad request: idempotency key missing",
			userID:     7,
			svc:        &fakeOrderService{},
			wantStatus: http.StatusBadRequest,
			wantCode:   "IDEMPOTENCY_KEY_REQUIRED",
		},
		{
			name:           "insufficient stock maps to 422",
			userID:         7,
			idempotencyKey: "key-stock",
			svc:            &fakeOrderService{createErr: validation.ErrInsufficientStock},
			wantStatus:     http.StatusUnprocessableEntity,
			wantCode:       "INSUFFICIENT_STOCK",
		},
		{
			name:           "unknown service error maps to 500",
			userID:         7,
			idempotencyKey: "key-error",
			svc:            &fakeOrderService{createErr: internalErr},
			wantStatus:     http.StatusInternalServerError,
			wantCode:       "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(nil, nil, nil, tt.svc, nil)

			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, createOrderRequest(tt.userID, tt.idempotencyKey))

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantCode != "" {
				var body ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != tt.wantCode {
					t.Errorf("error code = %q (decode err %v), want %q", body.Code, err, tt.wantCode)
				}
			}
		})
	}
}

func TestHandler_CreateOrder_ResponseContainsOrderAndPaymentLink(t *testing.T) {
	svc := &fakeOrderService{createResult: newCreateOrderResult()}
	h := newTestHandler(nil, nil, nil, svc, nil)

	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, createOrderRequest(7, "key-body"))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if svc.userID != 7 {
		t.Errorf("order created for user %d, want user from token (7)", svc.userID)
	}

	var body struct {
		Order         models.Order `json:"order"`
		XsollaOrderID int          `json:"xsolla_order_id"`
		PaymentURL    string       `json:"payment_url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body.Order.ID != 1 || body.Order.Status != "pending" {
		t.Errorf("order = %+v, want id 1 with status pending", body.Order)
	}
	if body.XsollaOrderID != 555 {
		t.Errorf("xsolla_order_id = %d, want 555", body.XsollaOrderID)
	}
	if body.PaymentURL != "https://sandbox-secure.xsolla.com/paystation4/?token=tok" {
		t.Errorf("payment_url = %q", body.PaymentURL)
	}
}

func TestHandler_CreateOrder_IdempotencyPreventsDuplicateServiceCall(
	t *testing.T,
) {
	svc := &fakeOrderService{createResult: newCreateOrderResult()}
	router := newTestHandler(nil, nil, nil, svc, nil).Routes()

	var bodies []string

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, createOrderRequest(7, "same-key"))

		if w.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d", i+1, w.Code, http.StatusCreated)
		}
		bodies = append(bodies, w.Body.String())
	}

	if svc.createCalls != 1 {
		t.Fatalf("CreateOrder called %d times, want 1", svc.createCalls)
	}
	if bodies[0] != bodies[1] {
		t.Errorf("replayed response differs from original:\n%s\n%s", bodies[0], bodies[1])
	}
}

func TestHandler_CreateOrder_FailedRequestIsNotCachedForIdempotency(t *testing.T) {
	// A transient failure must not "burn" the key: the client's retry with
	// the same key should reach the service again.
	svc := &fakeOrderService{createErr: errors.New("db down")}
	router := newTestHandler(nil, nil, nil, svc, nil).Routes()

	router.ServeHTTP(httptest.NewRecorder(), createOrderRequest(7, "retry-key"))

	svc.createErr = nil
	svc.createResult = newCreateOrderResult()

	w := httptest.NewRecorder()
	router.ServeHTTP(w, createOrderRequest(7, "retry-key"))

	if w.Code != http.StatusCreated || svc.createCalls != 2 {
		t.Fatalf("retry status = %d with %d service calls, want 201 and 2 calls", w.Code, svc.createCalls)
	}
}

func TestHandler_CreateOrder_IdempotencyKeyIsScopedToUser(t *testing.T) {
	// The cached response is for "the same request". A different user who
	// happens to send the same key is a different request and must not
	// receive another user's order or payment link.
	svc := &fakeOrderService{createResult: newCreateOrderResult()}
	router := newTestHandler(nil, nil, nil, svc, nil).Routes()

	router.ServeHTTP(httptest.NewRecorder(), createOrderRequest(7, "shared-key"))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, createOrderRequest(8, "shared-key"))

	if svc.createCalls != 2 {
		t.Fatalf("CreateOrder called %d times, want 2 (once per user); user 8 got user 7's cached response: %s",
			svc.createCalls, w.Body.String())
	}
	if svc.userID != 8 {
		t.Errorf("second order created for user %d, want 8", svc.userID)
	}
}

func TestHandler_GetUserOrders(t *testing.T) {
	internalErr := errors.New(
		"orders read failed",
	)

	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeOrderService
		wantStatus int
	}{
		{
			name:       "ok: returns orders",
			path:       "/user/orders?limit=10&offset=0",
			authorized: true,
			svc: &fakeOrderService{
				offsetResult: services.OrderOffsetResult{
					Orders: []models.UserOrder{
						{
							ID:     1,
							UserID: 7,
							Total:  1000,
							Status: "paid",
						},
					},
					Limit:  10,
					Offset: 0,
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			path:       "/user/orders",
			svc:        &fakeOrderService{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bad request: invalid limit",
			path:       "/user/orders?limit=abc",
			authorized: true,
			svc:        &fakeOrderService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request: negative offset",
			path:       "/user/orders?offset=-1",
			authorized: true,
			svc:        &fakeOrderService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown service error maps to 500",
			path:       "/user/orders",
			authorized: true,
			svc: &fakeOrderService{
				offsetErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(
				nil,
				nil,
				nil,
				tt.svc,
				nil,
			)

			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			if tt.authorized {
				req.Header.Set(
					"Authorization",
					"Bearer "+
						testAccessToken(7),
				)
			}

			w := httptest.NewRecorder()

			h.Routes().ServeHTTP(
				w,
				req,
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}
		})
	}
}

func TestHandler_GetUserOrdersCursor(
	t *testing.T,
) {
	internalErr := errors.New(
		"orders cursor failed",
	)

	tests := []struct {
		name       string
		path       string
		authorized bool
		svc        *fakeOrderService
		wantStatus int
	}{
		{
			name:       "ok: returns cursor page",
			path:       "/user/orders/cursor?limit=10",
			authorized: true,
			svc: &fakeOrderService{
				cursorResult: services.OrderCursorResult{
					Orders: []models.UserOrder{
						{
							ID: 1,
						},
					},
					HasMore: false,
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			path:       "/user/orders/cursor",
			svc:        &fakeOrderService{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bad request: invalid limit",
			path:       "/user/orders/cursor?limit=abc",
			authorized: true,
			svc:        &fakeOrderService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request: invalid cursor",
			path:       "/user/orders/cursor?cursor=bad",
			authorized: true,
			svc: &fakeOrderService{
				cursorErr: services.ErrInvalidCursor,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown service error maps to 500",
			path:       "/user/orders/cursor",
			authorized: true,
			svc: &fakeOrderService{
				cursorErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(
				nil,
				nil,
				nil,
				tt.svc,
				nil,
			)

			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			if tt.authorized {
				req.Header.Set(
					"Authorization",
					"Bearer "+
						testAccessToken(7),
				)
			}

			w := httptest.NewRecorder()

			h.Routes().ServeHTTP(
				w,
				req,
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}
		})
	}
}
