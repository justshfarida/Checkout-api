package services

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/validation"
)

// newTestOrderService wires OrderService with fakes. The legacy
// PaymentProcessor is no longer used by CreateOrder (payment now goes
// through Xsolla), so it is passed as a fake that must never be called.
func newTestOrderService(
	orderRepo *fakeOrderRepository,
	cartRepo *fakeCartRepository,
	itemRepo *fakeItemRepository,
	xsolla *fakeXsollaPaymentService,
	xsollaRepo *fakeXsollaRepository,
) (*OrderService, *fakePaymentProcessor) {
	payment := &fakePaymentProcessor{}

	return NewOrderService(
		orderRepo,
		cartRepo,
		itemRepo,
		payment,
		xsolla,
		xsollaRepo,
	), payment
}

func TestOrderService_CreateOrder(t *testing.T) {
	const userID = 7

	merch := func(id, price, stock int) *models.Item {
		return &models.Item{ID: id, Price: price, Stock: stock, ItemType: "merch"}
	}

	tests := []struct {
		name string

		cartItems []models.CartItem
		items     map[int]*models.Item

		cartErr   error
		itemErr   error
		createErr error
		xsollaErr error

		wantErr error

		wantTotal        int
		wantLineItems    []models.LineItem
		wantXsollaItems  []XsollaItem
		wantCreateCalled bool
		wantXsollaCalled bool
	}{
		{
			name:      "empty cart returns ErrEmptyCart and creates nothing",
			cartItems: []models.CartItem{},
			items:     map[int]*models.Item{},
			wantErr:   ErrEmptyCart,
		},
		{
			name:      "one item creates pending order and Xsolla payment",
			cartItems: []models.CartItem{{ID: 1, Quantity: 1}},
			items:     map[int]*models.Item{1: merch(1, 2500, 10)},

			wantTotal:        2500,
			wantLineItems:    []models.LineItem{{ItemID: 1, Quantity: 1, Price: 2500}},
			wantXsollaItems:  []XsollaItem{{SKU: "item_1", Quantity: 1}},
			wantCreateCalled: true,
			wantXsollaCalled: true,
		},
		{
			name: "many items: total uses database prices, SKUs follow item_<id>",
			cartItems: []models.CartItem{
				// Cart price is deliberately wrong; the DB price must win.
				{ID: 1, Quantity: 2, Price: 1},
				{ID: 2, Quantity: 3, Price: 1},
			},
			items: map[int]*models.Item{
				1: merch(1, 500, 10),
				2: merch(2, 1200, 10),
			},

			// 500*2 + 1200*3 = 4600
			wantTotal: 4600,
			wantLineItems: []models.LineItem{
				{ItemID: 1, Quantity: 2, Price: 500},
				{ItemID: 2, Quantity: 3, Price: 1200},
			},
			wantXsollaItems: []XsollaItem{
				{SKU: "item_1", Quantity: 2},
				{SKU: "item_2", Quantity: 3},
			},
			wantCreateCalled: true,
			wantXsollaCalled: true,
		},
		{
			name:      "merch quantity exactly equal to stock succeeds",
			cartItems: []models.CartItem{{ID: 1, Quantity: 5}},
			items:     map[int]*models.Item{1: merch(1, 300, 5)},

			wantTotal:        1500,
			wantLineItems:    []models.LineItem{{ItemID: 1, Quantity: 5, Price: 300}},
			wantXsollaItems:  []XsollaItem{{SKU: "item_1", Quantity: 5}},
			wantCreateCalled: true,
			wantXsollaCalled: true,
		},
		{
			name:      "merch quantity greater than stock returns insufficient stock",
			cartItems: []models.CartItem{{ID: 1, Quantity: 6}},
			items:     map[int]*models.Item{1: merch(1, 300, 5)},
			wantErr:   validation.ErrInsufficientStock,
		},
		{
			name:      "non-merch item is not limited by stock",
			cartItems: []models.CartItem{{ID: 3, Quantity: 50}},
			items: map[int]*models.Item{
				3: {ID: 3, Price: 100, Stock: 0, ItemType: "currency"},
			},

			wantTotal:        5000,
			wantLineItems:    []models.LineItem{{ItemID: 3, Quantity: 50, Price: 100}},
			wantXsollaItems:  []XsollaItem{{SKU: "item_3", Quantity: 50}},
			wantCreateCalled: true,
			wantXsollaCalled: true,
		},
		{
			name:      "missing item returns ErrItemNotFound",
			cartItems: []models.CartItem{{ID: 999, Quantity: 1}},
			items:     map[int]*models.Item{},
			wantErr:   ErrItemNotFound,
		},
		{
			name:    "cart repository error is returned",
			cartErr: errFakeDependency,
			wantErr: errFakeDependency,
		},
		{
			name:      "item repository error is returned",
			cartItems: []models.CartItem{{ID: 1, Quantity: 1}},
			itemErr:   errFakeDependency,
			wantErr:   errFakeDependency,
		},
		{
			name:      "order repository error stops before Xsolla is called",
			cartItems: []models.CartItem{{ID: 1, Quantity: 1}},
			items:     map[int]*models.Item{1: merch(1, 1000, 10)},
			createErr: errFakeDependency,

			wantErr:          errFakeDependency,
			wantCreateCalled: true,
		},
		{
			name:      "Xsolla error is returned",
			cartItems: []models.CartItem{{ID: 1, Quantity: 1}},
			items:     map[int]*models.Item{1: merch(1, 1000, 10)},
			xsollaErr: errFakeDependency,

			wantErr:          errFakeDependency,
			wantCreateCalled: true,
			wantXsollaCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cartRepo := &fakeCartRepository{items: tt.cartItems, getErr: tt.cartErr}
			itemRepo := &fakeItemRepository{items: tt.items, err: tt.itemErr}
			orderRepo := &fakeOrderRepository{createErr: tt.createErr}
			xsolla := &fakeXsollaPaymentService{
				response: &XsollaPaymentResponse{Token: "tok_abc", OrderID: 555},
				err:      tt.xsollaErr,
			}

			service, payment := newTestOrderService(
				orderRepo, cartRepo, itemRepo, xsolla, &fakeXsollaRepository{},
			)

			got, err := service.CreateOrder(context.Background(), userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateOrder() error = %v, want %v", err, tt.wantErr)
			}
			if orderRepo.createCalled != tt.wantCreateCalled {
				t.Errorf("CreateOrder repository called = %v, want %v",
					orderRepo.createCalled, tt.wantCreateCalled)
			}
			if xsolla.called != tt.wantXsollaCalled {
				t.Errorf("Xsolla CreatePayment called = %v, want %v",
					xsolla.called, tt.wantXsollaCalled)
			}
			if payment.called {
				t.Error("legacy payment processor must not be called; payment is confirmed by the Xsolla webhook")
			}

			if tt.wantErr != nil {
				if got != nil {
					t.Errorf("result = %#v, want nil on error", got)
				}
				return
			}

			if got == nil || got.Order == nil {
				t.Fatal("CreateOrder() returned nil result/order")
			}
			if orderRepo.createdStatus != "pending" {
				t.Errorf("order created with status %q, want pending", orderRepo.createdStatus)
			}
			if orderRepo.createdUserID != userID {
				t.Errorf("order created for user %d, want %d", orderRepo.createdUserID, userID)
			}
			if orderRepo.createdTotal != tt.wantTotal {
				t.Errorf("order total = %d, want %d", orderRepo.createdTotal, tt.wantTotal)
			}
			if !reflect.DeepEqual(orderRepo.createdItems, tt.wantLineItems) {
				t.Errorf("line items = %+v, want %+v", orderRepo.createdItems, tt.wantLineItems)
			}
			if !reflect.DeepEqual(xsolla.items, tt.wantXsollaItems) {
				t.Errorf("Xsolla items = %+v, want %+v", xsolla.items, tt.wantXsollaItems)
			}
			if xsolla.userID != userID || xsolla.orderID != got.Order.ID {
				t.Errorf("Xsolla called with user=%d order=%d, want user=%d order=%d",
					xsolla.userID, xsolla.orderID, userID, got.Order.ID)
			}
			if got.XsollaOrderID != 555 {
				t.Errorf("XsollaOrderID = %d, want 555", got.XsollaOrderID)
			}
			wantURL := "https://sandbox-secure.xsolla.com/paystation4/?token=tok_abc"
			if got.PaymentURL != wantURL {
				t.Errorf("PaymentURL = %q, want %q", got.PaymentURL, wantURL)
			}
		})
	}
}

func TestOrderService_ProcessXsollaPayment(t *testing.T) {
	const orderID = 42
	const transactionID int64 = 9_000_000_001 // larger than int32

	lineItems := []models.LineItem{{ItemID: 1, Quantity: 2, Price: 500}}

	tests := []struct {
		name string

		orderItems    []models.LineItem
		orderItemsErr error
		processErr    error

		wantErr        error
		wantRepoCalled bool
	}{
		{
			name:           "passes order items and transaction to repository",
			orderItems:     lineItems,
			wantRepoCalled: true,
		},
		{
			name:       "order without items returns ErrOrderNotFound",
			orderItems: []models.LineItem{},
			wantErr:    ErrOrderNotFound,
		},
		{
			name:          "GetOrderItems error is returned",
			orderItemsErr: errFakeDependency,
			wantErr:       errFakeDependency,
		},
		{
			name:           "repository processing error is returned",
			orderItems:     lineItems,
			processErr:     errFakeDependency,
			wantErr:        errFakeDependency,
			wantRepoCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &fakeOrderRepository{orderItems: tt.orderItems, orderItemsErr: tt.orderItemsErr}
			xsollaRepo := &fakeXsollaRepository{processed: true, err: tt.processErr}

			service, _ := newTestOrderService(
				orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
				&fakeXsollaPaymentService{}, xsollaRepo,
			)

			err := service.ProcessXsollaPayment(context.Background(), orderID, transactionID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ProcessXsollaPayment() error = %v, want %v", err, tt.wantErr)
			}
			if orderRepo.orderItemsOrderID != orderID {
				t.Errorf("GetOrderItems orderID = %d, want %d", orderRepo.orderItemsOrderID, orderID)
			}
			if xsollaRepo.called != tt.wantRepoCalled {
				t.Fatalf("xsolla repository called = %v, want %v", xsollaRepo.called, tt.wantRepoCalled)
			}
			if !tt.wantRepoCalled {
				return
			}
			if xsollaRepo.orderID != orderID || xsollaRepo.transactionID != transactionID {
				t.Errorf("processed order=%d txn=%d, want order=%d txn=%d",
					xsollaRepo.orderID, xsollaRepo.transactionID, orderID, transactionID)
			}
			if !reflect.DeepEqual(xsollaRepo.items, tt.orderItems) {
				t.Errorf("processed items = %+v, want %+v", xsollaRepo.items, tt.orderItems)
			}
		})
	}
}

func TestOrderService_ProcessXsollaPayment_DuplicateWebhookIsNotAnError(t *testing.T) {
	// Xsolla retries webhooks; a transaction that was already processed
	// (repository returns processed=false) must still be acknowledged.
	orderRepo := &fakeOrderRepository{orderItems: []models.LineItem{{ItemID: 1, Quantity: 1, Price: 100}}}
	xsollaRepo := &fakeXsollaRepository{processed: false}

	service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
		&fakeXsollaPaymentService{}, xsollaRepo)

	if err := service.ProcessXsollaPayment(context.Background(), 1, 77); err != nil {
		t.Fatalf("duplicate webhook returned error %v, want nil", err)
	}
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	t.Run("forwards order ID and status", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{}
		service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		if err := service.UpdateOrderStatus(context.Background(), 12, "canceled"); err != nil {
			t.Fatalf("UpdateOrderStatus() error = %v", err)
		}
		if orderRepo.updatedOrderID != 12 || orderRepo.updatedStatus != "canceled" {
			t.Errorf("updated order=%d status=%q, want order=12 status=canceled",
				orderRepo.updatedOrderID, orderRepo.updatedStatus)
		}
	})

	t.Run("returns repository error", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{updateErr: errFakeDependency}
		service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		if err := service.UpdateOrderStatus(context.Background(), 12, "canceled"); !errors.Is(err, errFakeDependency) {
			t.Fatalf("UpdateOrderStatus() error = %v, want %v", err, errFakeDependency)
		}
	})
}

func TestOrderService_GetOrdersByUserIDOffset(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		repoErr    error
		wantLimit  int
		wantOffset int
		wantErr    error
	}{
		{name: "zero limit uses default", limit: 0, offset: 0, wantLimit: DefaultLimit, wantOffset: 0},
		{name: "limit above max is capped", limit: 1000, offset: 5, wantLimit: MaxLimit, wantOffset: 5},
		{name: "negative offset becomes zero", limit: 5, offset: -3, wantLimit: 5, wantOffset: 0},
		{name: "repository error is returned", limit: 5, repoErr: errFakeDependency, wantErr: errFakeDependency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := []models.UserOrder{{ID: 1, UserID: 3}}
			orderRepo := &fakeOrderRepository{userOrders: orders, userOrdersErr: tt.repoErr}
			service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
				&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

			got, err := service.GetOrdersByUserIDOffset(context.Background(), 3, tt.limit, tt.offset)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if orderRepo.listUserID != 3 || orderRepo.listLimit != tt.wantLimit || orderRepo.listOffset != tt.wantOffset {
				t.Errorf("repo called with user=%d limit=%d offset=%d, want user=3 limit=%d offset=%d",
					orderRepo.listUserID, orderRepo.listLimit, orderRepo.listOffset, tt.wantLimit, tt.wantOffset)
			}
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset || !reflect.DeepEqual(got.Orders, orders) {
				t.Errorf("result = %+v, want limit=%d offset=%d orders=%+v", got, tt.wantLimit, tt.wantOffset, orders)
			}
		})
	}
}

func TestOrderService_GetOrdersByUserIDCursor(t *testing.T) {
	ordersN := func(n int) []models.UserOrder {
		out := make([]models.UserOrder, n)
		for i := range out {
			out[i] = models.UserOrder{ID: i + 1}
		}
		return out
	}

	t.Run("more rows than limit: trims page and returns cursor of last returned order", func(t *testing.T) {
		// Repository is asked for limit+1 rows to detect another page.
		orderRepo := &fakeOrderRepository{userOrders: ordersN(3)}
		service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		got, err := service.GetOrdersByUserIDCursor(context.Background(), 1, "", 2)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if orderRepo.listLimit != 3 {
			t.Errorf("repository limit = %d, want 3 (limit+1)", orderRepo.listLimit)
		}
		if len(got.Orders) != 2 || !got.HasMore {
			t.Fatalf("got %d orders hasMore=%v, want 2 orders hasMore=true", len(got.Orders), got.HasMore)
		}
		lastID, err := DecodeCursor(got.NextCursor)
		if err != nil || lastID != 2 {
			t.Errorf("next cursor decodes to %d (err %v), want 2", lastID, err)
		}
	})

	t.Run("last page has no cursor", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{userOrders: ordersN(2)}
		service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		got, err := service.GetOrdersByUserIDCursor(context.Background(), 1, EncodeCursor(5), 2)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if orderRepo.listCursorLastID != 5 {
			t.Errorf("repository lastID = %d, want 5", orderRepo.listCursorLastID)
		}
		if got.HasMore || got.NextCursor != "" || len(got.Orders) != 2 {
			t.Errorf("got %+v, want 2 orders, hasMore=false, empty cursor", got)
		}
	})

	t.Run("invalid cursor returns ErrInvalidCursor without querying", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{}
		service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		_, err := service.GetOrdersByUserIDCursor(context.Background(), 1, "%%%not-base64", 2)
		if !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("error = %v, want ErrInvalidCursor", err)
		}
		if orderRepo.listLimit != 0 {
			t.Error("repository was queried despite invalid cursor")
		}
	})

	t.Run("repository error is returned", func(t *testing.T) {
		orderRepo := &fakeOrderRepository{userOrdersErr: errFakeDependency}
		service, _ := newTestOrderService(orderRepo, &fakeCartRepository{}, &fakeItemRepository{},
			&fakeXsollaPaymentService{}, &fakeXsollaRepository{})

		if _, err := service.GetOrdersByUserIDCursor(context.Background(), 1, "", 2); !errors.Is(err, errFakeDependency) {
			t.Fatalf("error = %v, want %v", err, errFakeDependency)
		}
	})
}
