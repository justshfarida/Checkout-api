package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// signXsolla builds the Authorization header exactly as Xsolla documents it:
// "Signature " + hex(SHA-1(rawBody + secretKey)).
func signXsolla(body []byte, secret string) string {
	sum := sha1.Sum(append(append([]byte{}, body...), []byte(secret)...))
	return "Signature " + hex.EncodeToString(sum[:])
}

func TestXsollaService_VerifyXsollaSignature(t *testing.T) {
	const secret = "webhook-secret"
	body := []byte(`{"notification_type":"order_paid","custom_parameters":{"order_id":"5"}}`)

	service := NewXsollaService("proj", "merchant", "api-key", secret, "")

	tests := []struct {
		name   string
		body   []byte
		header string
		want   bool
	}{
		{
			name:   "valid signature is accepted",
			body:   body,
			header: signXsolla(body, secret),
			want:   true,
		},
		{
			name:   "signature made with a different secret is rejected",
			body:   body,
			header: signXsolla(body, "attacker-secret"),
			want:   false,
		},
		{
			name:   "tampered body is rejected",
			body:   []byte(strings.Replace(string(body), `"5"`, `"6"`, 1)),
			header: signXsolla(body, secret),
			want:   false,
		},
		{
			name:   "header without Signature scheme is rejected",
			body:   body,
			header: strings.TrimPrefix(signXsolla(body, secret), "Signature "),
			want:   false,
		},
		{
			name:   "empty header is rejected",
			body:   body,
			header: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.VerifyXsollaSignature(tt.body, tt.header); got != tt.want {
				t.Errorf("VerifyXsollaSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

// roundTripFunc lets a test intercept outgoing HTTP calls without network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestXsollaService_CreatePayment_SendsDocumentedRequest(t *testing.T) {
	service := NewXsollaService("12345", "merchant", "secret-api-key", "wh", "http://checkout.local/payment/success")

	var captured *http.Request
	var capturedBody []byte

	service.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		captured = r
		capturedBody, _ = io.ReadAll(r.Body)
		return jsonResponse(http.StatusOK, `{"token":"tok_1","order_id":987}`), nil
	})}

	items := []XsollaItem{{SKU: "item_1", Quantity: 2}}

	got, err := service.CreatePayment(context.Background(), 7, 42, items)
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}

	if got.Token != "tok_1" || got.OrderID != 987 {
		t.Errorf("response = %+v, want token tok_1 and order 987", got)
	}

	if captured.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", captured.Method)
	}
	wantURL := "https://store.xsolla.com/api/v3/project/12345/admin/payment/token"
	if captured.URL.String() != wantURL {
		t.Errorf("url = %s, want %s", captured.URL, wantURL)
	}
	user, pass, ok := captured.BasicAuth()
	if !ok || user != "12345" || pass != "secret-api-key" {
		t.Errorf("basic auth = (%q, %q, %v), want project ID and API key", user, pass, ok)
	}

	var sent struct {
		Sandbox bool `json:"sandbox"`
		User    struct {
			ID struct {
				Value string `json:"value"`
			} `json:"id"`
		} `json:"user"`
		Purchase struct {
			Items []XsollaItem `json:"items"`
		} `json:"purchase"`
		CustomParameters map[string]string `json:"custom_parameters"`
		Settings         struct {
			ReturnURL string `json:"return_url"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(capturedBody, &sent); err != nil {
		t.Fatalf("request body is not JSON: %v", err)
	}
	if sent.User.ID.Value != "7" {
		t.Errorf("user.id.value = %q, want 7", sent.User.ID.Value)
	}
	if len(sent.Purchase.Items) != 1 || sent.Purchase.Items[0] != items[0] {
		t.Errorf("purchase.items = %+v, want %+v", sent.Purchase.Items, items)
	}
	// The webhook handler reads custom_parameters.order_id to find the order.
	if sent.CustomParameters["order_id"] != "42" {
		t.Errorf("custom_parameters.order_id = %q, want 42", sent.CustomParameters["order_id"])
	}
	// Each environment must redirect back to its own frontend after payment.
	if sent.Settings.ReturnURL != "http://checkout.local/payment/success" {
		t.Errorf("settings.return_url = %q, want this environment's success page", sent.Settings.ReturnURL)
	}
}

func TestXsollaService_CreatePayment_NonSuccessStatusReturnsError(t *testing.T) {
	service := NewXsollaService("12345", "merchant", "key", "wh", "")
	service.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnprocessableEntity, `{"errorCode":1102,"errorMessage":"Item not found"}`), nil
	})}

	got, err := service.CreatePayment(context.Background(), 1, 1, nil)

	if err == nil {
		t.Fatal("CreatePayment() error = nil, want error for HTTP 422")
	}
	if got != nil {
		t.Errorf("response = %+v, want nil on error", got)
	}
	if !strings.Contains(err.Error(), "Item not found") {
		t.Errorf("error %q should include Xsolla's message for debugging", err)
	}
}
