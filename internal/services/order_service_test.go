package services

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/validation"
)

func TestOrderService_CreateOrder(
	t *testing.T,
) {
	const userID = 7

	tests := []struct {
		name string

		cartItems []models.CartItem
		items     map[int]*models.Item

		cartErr     error
		itemErr     error
		createErr   error
		updateErr   error
		markPaidErr error
		deleteErr   error
		paymentErr  error

		paymentResult PaymentResult

		wantErr            error
		wantStatus         string
		wantTotal          int
		wantPaymentSuccess bool

		wantCreateCalled   bool
		wantPaymentCalled  bool
		wantUpdateCalled   bool
		wantMarkPaidCalled bool
		wantCartCleared    bool
	}{
		{
			name: "empty cart returns ErrEmptyCart",

			cartItems: []models.CartItem{},
			items:     map[int]*models.Item{},

			wantErr: ErrEmptyCart,
		},
		{
			name: "one item payment succeeds",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 2500,
					Stock: 10,
				},
			},

			paymentResult: PaymentResult{
				Success:       true,
				TransactionID: "txn_test",
			},

			wantStatus:         "paid",
			wantTotal:          2500,
			wantPaymentSuccess: true,

			wantCreateCalled:   true,
			wantPaymentCalled:  true,
			wantMarkPaidCalled: true,
			wantCartCleared:    true,
		},
		{
			name: "many items calculates total from database prices",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 2,
				},
				{
					ID:       2,
					Quantity: 3,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 500,
					Stock: 10,
				},
				2: {
					ID:    2,
					Price: 1200,
					Stock: 10,
				},
			},

			paymentResult: PaymentResult{
				Success: true,
			},

			// 500*2 + 1200*3 = 4600
			wantTotal:          4600,
			wantStatus:         "paid",
			wantPaymentSuccess: true,

			wantCreateCalled:   true,
			wantPaymentCalled:  true,
			wantMarkPaidCalled: true,
			wantCartCleared:    true,
		},
		{
			name: "quantity exactly equal to stock succeeds",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 5,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 300,
					Stock: 5,
				},
			},

			paymentResult: PaymentResult{
				Success: true,
			},

			wantStatus:         "paid",
			wantTotal:          1500,
			wantPaymentSuccess: true,

			wantCreateCalled:   true,
			wantPaymentCalled:  true,
			wantMarkPaidCalled: true,
			wantCartCleared:    true,
		},
		{
			name: "quantity greater than stock returns insufficient stock",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 6,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 300,
					Stock: 5,
				},
			},

			wantErr: validation.ErrInsufficientStock,
		},
		{
			name: "missing item returns ErrItemNotFound",

			cartItems: []models.CartItem{
				{
					ID:       999,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{},

			wantErr: ErrItemNotFound,
		},
		{
			name: "cart repository error is returned",

			cartErr: errFakeDependency,

			items: map[int]*models.Item{},

			wantErr: errFakeDependency,
		},
		{
			name: "item repository error is returned",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{},

			itemErr: errFakeDependency,

			wantErr: errFakeDependency,
		},
		{
			name: "order repository create error is returned",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 1000,
					Stock: 10,
				},
			},

			createErr: errFakeDependency,

			wantErr:          errFakeDependency,
			wantCreateCalled: true,
		},
		{
			name: "payment declined marks order failed and keeps cart",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 2500,
					Stock: 10,
				},
			},

			paymentResult: PaymentResult{
				Success: false,
				Error:   "Payment declined",
			},

			wantStatus:         "failed",
			wantTotal:          2500,
			wantPaymentSuccess: false,

			wantCreateCalled:   true,
			wantPaymentCalled:  true,
			wantUpdateCalled:   true,
			wantMarkPaidCalled: false,
			wantCartCleared:    false,
		},
		{
			name: "payment processor error is returned",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 2500,
					Stock: 10,
				},
			},

			paymentErr: errFakeDependency,

			wantErr: errFakeDependency,

			wantCreateCalled:  true,
			wantPaymentCalled: true,
		},
		{
			name: "mark paid repository error is returned",

			cartItems: []models.CartItem{
				{
					ID:       1,
					Quantity: 1,
				},
			},

			items: map[int]*models.Item{
				1: {
					ID:    1,
					Price: 2500,
					Stock: 10,
				},
			},

			paymentResult: PaymentResult{
				Success: true,
			},

			markPaidErr: errFakeDependency,

			wantErr: errFakeDependency,

			wantCreateCalled:   true,
			wantPaymentCalled:  true,
			wantMarkPaidCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				// ARRANGE

				cartRepo :=
					&fakeCartRepository{
						items:     tt.cartItems,
						getErr:    tt.cartErr,
						deleteErr: tt.deleteErr,
					}

				itemRepo :=
					&fakeItemRepository{
						items: tt.items,
						err:   tt.itemErr,
					}

				orderRepo :=
					&fakeOrderRepository{
						createErr:   tt.createErr,
						updateErr:   tt.updateErr,
						markPaidErr: tt.markPaidErr,
					}

				payment :=
					&fakePaymentProcessor{
						result: tt.paymentResult,
						err:    tt.paymentErr,
					}

				service :=
					NewOrderService(
						orderRepo,
						cartRepo,
						itemRepo,
						payment,
					)

				// ACT

				got, err :=
					service.CreateOrder(
						context.Background(),
						userID,
					)

				// ASSERT ERROR

				if !errors.Is(
					err,
					tt.wantErr,
				) {
					t.Fatalf(
						"CreateOrder() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				// ASSERT SIDE EFFECTS

				if orderRepo.createCalled !=
					tt.wantCreateCalled {

					t.Errorf(
						"CreateOrder repository called = %v, want %v",
						orderRepo.createCalled,
						tt.wantCreateCalled,
					)
				}

				if payment.called !=
					tt.wantPaymentCalled {

					t.Errorf(
						"payment called = %v, want %v",
						payment.called,
						tt.wantPaymentCalled,
					)
				}

				if orderRepo.updateCalled !=
					tt.wantUpdateCalled {

					t.Errorf(
						"UpdateOrderStatus called = %v, want %v",
						orderRepo.updateCalled,
						tt.wantUpdateCalled,
					)
				}

				if orderRepo.markPaidCalled !=
					tt.wantMarkPaidCalled {

					t.Errorf(
						"MarkOrderPaid called = %v, want %v",
						orderRepo.markPaidCalled,
						tt.wantMarkPaidCalled,
					)
				}

				if cartRepo.deleteCalled !=
					tt.wantCartCleared {

					t.Errorf(
						"cart cleared = %v, want %v",
						cartRepo.deleteCalled,
						tt.wantCartCleared,
					)
				}

				// Error paths should return no result.
				if tt.wantErr != nil {
					if got != nil {
						t.Errorf(
							"result = %#v, want nil",
							got,
						)
					}

					return
				}

				// ASSERT RESULT

				if got == nil {
					t.Fatal(
						"CreateOrder() returned nil result",
					)
				}

				if got.Order == nil {
					t.Fatal(
						"CreateOrder() returned nil order",
					)
				}

				if got.Order.Status !=
					tt.wantStatus {

					t.Errorf(
						"order status = %q, want %q",
						got.Order.Status,
						tt.wantStatus,
					)
				}

				if got.Payment.Success !=
					tt.wantPaymentSuccess {

					t.Errorf(
						"payment success = %v, want %v",
						got.Payment.Success,
						tt.wantPaymentSuccess,
					)
				}

				if orderRepo.createdTotal !=
					tt.wantTotal {

					t.Errorf(
						"order total = %d, want %d",
						orderRepo.createdTotal,
						tt.wantTotal,
					)
				}

				if payment.amount !=
					tt.wantTotal {

					t.Errorf(
						"payment amount = %d, want %d",
						payment.amount,
						tt.wantTotal,
					)
				}

				if orderRepo.createdStatus !=
					"pending" {

					t.Errorf(
						"initial order status = %q, want pending",
						orderRepo.createdStatus,
					)
				}

				if tt.wantUpdateCalled &&
					orderRepo.updatedStatus !=
						"failed" {

					t.Errorf(
						"updated status = %q, want failed",
						orderRepo.updatedStatus,
					)
				}

				if tt.wantCartCleared &&
					cartRepo.deletedUserID !=
						userID {

					t.Errorf(
						"deleted cart userID = %d, want %d",
						cartRepo.deletedUserID,
						userID,
					)
				}
			},
		)
	}
}
