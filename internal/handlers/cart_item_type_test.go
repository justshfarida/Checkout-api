package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/services"
)

// The frontend rebuilds the cart from these endpoints after a full page
// load (e.g. coming back from Pay Station) and decides from item_type
// whether stock applies. Dropping the field hides stock for merch items.
func TestHandler_CartResponsesIncludeItemTypeAndCreditAmount(t *testing.T) {
	gold := 500
	items := []models.CartItem{
		{ID: 1, Name: "T-Shirt", Stock: 7, Quantity: 1, ItemType: "merch"},
		{ID: 2, Name: "500 Gold", Quantity: 1, ItemType: "currency_package", CreditAmount: &gold},
	}

	svc := &fakeCartService{
		offsetResult: services.CartOffsetResult{Items: items, Limit: 10, Total: 2},
		cursorResult: services.CartCursorResult{Items: items, Total: 2},
	}
	router := newTestHandler(nil, nil, svc, nil, nil).Routes()

	for _, path := range []string{"/user/cart", "/user/cart/cursor"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+testAccessToken(7))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
			}

			var body struct {
				Items []struct {
					ItemType     string `json:"item_type"`
					Stock        int    `json:"stock"`
					CreditAmount *int   `json:"credit_amount"`
				} `json:"items"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Items) != 2 {
				t.Fatalf("decode: err=%v items=%d body=%s", err, len(body.Items), w.Body.String())
			}

			if body.Items[0].ItemType != "merch" || body.Items[0].Stock != 7 {
				t.Errorf("merch item = %+v, want item_type merch and stock 7", body.Items[0])
			}
			if body.Items[1].ItemType != "currency_package" ||
				body.Items[1].CreditAmount == nil || *body.Items[1].CreditAmount != 500 {
				t.Errorf("gold item = %+v, want item_type currency_package and credit_amount 500", body.Items[1])
			}
		})
	}
}
