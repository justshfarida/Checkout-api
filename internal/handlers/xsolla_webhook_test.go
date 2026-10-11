package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const webhookAuthHeader = "Signature abc123"

func webhookRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("Authorization", webhookAuthHeader)
	return req
}

func serveWebhook(
	t *testing.T,
	auth *fakeAuthService,
	orders *fakeOrderService,
	xsolla *fakeXsollaService,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	h := newTestHandlerWithXsolla(auth, nil, nil, orders, nil, xsolla)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, webhookRequest(body))

	return w
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()

	var body ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not an error JSON: %v; body=%s", err, w.Body.String())
	}
	return body.Code
}

func TestXsollaWebhook_InvalidSignatureIsRejectedBeforeAnyProcessing(t *testing.T) {
	body := `{"notification_type":"order_paid","custom_parameters":{"order_id":"5"},"billing":{"transaction":{"id":1}}}`
	orders := &fakeOrderService{}
	xsolla := &fakeXsollaService{signatureValid: false}

	w := serveWebhook(t, &fakeAuthService{}, orders, xsolla, body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if orders.processCalled {
		t.Error("payment was processed despite an invalid signature")
	}
	// The signature must be checked over the exact raw body and header.
	if string(xsolla.verifiedBody) != body || xsolla.verifiedAuth != webhookAuthHeader {
		t.Errorf("verified body=%q header=%q, want raw body and %q",
			xsolla.verifiedBody, xsolla.verifiedAuth, webhookAuthHeader)
	}
}

func TestXsollaWebhook_MalformedJSONWithValidSignatureReturns400(t *testing.T) {
	w := serveWebhook(t, &fakeAuthService{}, &fakeOrderService{}, &fakeXsollaService{signatureValid: true}, `{not json`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestXsollaWebhook_OrderPaid(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		processErr  error
		wantStatus  int
		wantProcess bool
		wantCode    string
	}{
		{
			name:        "processes payment for order from custom_parameters",
			body:        `{"notification_type":"order_paid","custom_parameters":{"order_id":"42"},"billing":{"transaction":{"id":9000000001}}}`,
			wantStatus:  http.StatusNoContent,
			wantProcess: true,
		},
		{
			name:       "non-numeric order id returns 400",
			body:       `{"notification_type":"order_paid","custom_parameters":{"order_id":"abc"},"billing":{"transaction":{"id":1}}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_ORDER_ID",
		},
		{
			// A 5xx makes Xsolla retry the webhook later.
			name:        "processing failure returns 500",
			body:        `{"notification_type":"order_paid","custom_parameters":{"order_id":"42"},"billing":{"transaction":{"id":1}}}`,
			processErr:  errors.New("db down"),
			wantStatus:  http.StatusInternalServerError,
			wantProcess: true,
			wantCode:    "PAYMENT_PROCESSING_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := &fakeOrderService{processErr: tt.processErr}

			w := serveWebhook(t, &fakeAuthService{}, orders, &fakeXsollaService{signatureValid: true}, tt.body)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if orders.processCalled != tt.wantProcess {
				t.Fatalf("ProcessXsollaPayment called = %v, want %v", orders.processCalled, tt.wantProcess)
			}
			if tt.wantCode != "" && errorCode(t, w) != tt.wantCode {
				t.Errorf("error code = %q, want %q", errorCode(t, w), tt.wantCode)
			}
			if tt.wantStatus == http.StatusNoContent {
				if orders.processOrderID != 42 || orders.processTransactionID != 9000000001 {
					t.Errorf("processed order=%d txn=%d, want order=42 txn=9000000001",
						orders.processOrderID, orders.processTransactionID)
				}
			}
		})
	}
}

func TestXsollaWebhook_OrderCanceled(t *testing.T) {
	t.Run("marks the order canceled", func(t *testing.T) {
		orders := &fakeOrderService{}
		body := `{"notification_type":"order_canceled","custom_parameters":{"order_id":"17"}}`

		w := serveWebhook(t, &fakeAuthService{}, orders, &fakeXsollaService{signatureValid: true}, body)

		if w.Code < 200 || w.Code > 299 {
			t.Fatalf("status = %d, want 2xx", w.Code)
		}
		if orders.updateOrderID != 17 || orders.updateStatus != "canceled" {
			t.Errorf("updated order=%d status=%q, want 17/canceled", orders.updateOrderID, orders.updateStatus)
		}
	})

	t.Run("update failure returns 500", func(t *testing.T) {
		orders := &fakeOrderService{updateErr: errors.New("db down")}
		body := `{"notification_type":"order_canceled","custom_parameters":{"order_id":"17"}}`

		w := serveWebhook(t, &fakeAuthService{}, orders, &fakeXsollaService{signatureValid: true}, body)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})
}

func TestXsollaWebhook_UserValidation(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		exists     bool
		existsErr  error
		wantStatus int
		wantCode   string
	}{
		{name: "existing user is valid", userID: "7", exists: true, wantStatus: http.StatusOK},
		{name: "unknown user returns INVALID_USER", userID: "7", exists: false, wantStatus: http.StatusBadRequest, wantCode: "INVALID_USER"},
		{name: "non-numeric user id returns INVALID_USER", userID: "abc", wantStatus: http.StatusBadRequest, wantCode: "INVALID_USER"},
		{
			// The handler labels this INTERNAL_ERROR; everywhere else in the
			// API that code means HTTP 500, and a 5xx lets Xsolla retry.
			name: "lookup failure is an internal error (500)", userID: "7",
			existsErr: errors.New("db down"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuthService{userExists: tt.exists, userExistsErr: tt.existsErr}
			body := `{"notification_type":"user_validation","user":{"id":"` + tt.userID + `"}}`

			w := serveWebhook(t, auth, &fakeOrderService{}, &fakeXsollaService{signatureValid: true}, body)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantCode != "" && errorCode(t, w) != tt.wantCode {
				t.Errorf("error code = %q, want %q", errorCode(t, w), tt.wantCode)
			}
		})
	}
}

func TestXsollaWebhook_UnknownNotificationIsAcknowledged(t *testing.T) {
	orders := &fakeOrderService{}

	w := serveWebhook(t, &fakeAuthService{}, orders, &fakeXsollaService{signatureValid: true},
		`{"notification_type":"refund"}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if orders.processCalled || orders.updateCalled {
		t.Error("unknown notification must not change orders")
	}
}
