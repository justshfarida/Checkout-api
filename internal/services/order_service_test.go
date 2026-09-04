package services

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
	"checkout-api/internal/validation"
)

var errFakeRepository = errors.New("fake repository error")

// -----------------------------------------------------------------------------
// Fake Item Repository
// -----------------------------------------------------------------------------

type fakeItemRepository struct {
	items map[int]*models.Item
	err   error
}

func (f *fakeItemRepository) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	if f.err != nil {
		return nil, f.err
	}

	item, ok := f.items[id]
	if !ok {
		return nil, nil
	}

	return item, nil
}

func (f *fakeItemRepository) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) ([]*models.Item, error) {
	return nil, nil
}

func (f *fakeItemRepository) GetItemsCursor(
	ctx context.Context,
	lastID int,
	limit int,
) ([]*models.Item, error) {
	return nil, nil
}

func (f *fakeItemRepository) CountItems(
	ctx context.Context,
) (int, error) {
	return len(f.items), nil
}

// Compile-time check.
var _ repositories.ItemRepository = (*fakeItemRepository)(nil)

// -----------------------------------------------------------------------------
// Fake Order Repository
// -----------------------------------------------------------------------------

type fakeOrderRepository struct {
	createErr error
	updateErr error

	createCalled bool
	updateCalled bool

	createdStatus string
	updatedStatus string
}

func (f *fakeOrderRepository) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	f.createCalled = true
	f.createdStatus = status

	if f.createErr != nil {
		return nil, f.createErr
	}

	return &models.Order{
		ID:     1,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}, nil
}

func (f *fakeOrderRepository) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	f.updateCalled = true
	f.updatedStatus = status

	if f.updateErr != nil {
		return f.updateErr
	}

	return nil
}

func (f *fakeOrderRepository) GetOrdersByUserIDOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.UserOrder, error) {
	return nil, nil
}

func (f *fakeOrderRepository) GetOrdersByUserIDCursor(
	ctx context.Context,
	userID int,
	lastID int,
	limit int,
) ([]models.UserOrder, error) {
	return nil, nil
}

// Compile-time check.
var _ repositories.OrderRepository = (*fakeOrderRepository)(nil)

// -----------------------------------------------------------------------------
// CreateOrder Tests
// -----------------------------------------------------------------------------

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name string

		itemsInRepo map[int]*models.Item
		lineItems   []models.LineItem
		total       int

		itemRepoErr   error
		createRepoErr error
		updateRepoErr error

		wantErr            error
		wantStatus         string
		wantPaymentSuccess bool
		wantCreateCalled   bool
		wantUpdateCalled   bool
	}{
		{
			name: "one item payment succeeds",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Name:  "T-Shirt",
					Price: 2500,
					Stock: 10,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
					Price:    2500,
				},
			},

			total: 2500,

			wantStatus:         "paid",
			wantPaymentSuccess: true,
			wantCreateCalled:   true,
			wantUpdateCalled:   true,
		},
		{
			name: "many items payment succeeds",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Name:  "Sticker",
					Price: 500,
					Stock: 20,
				},
				2: {
					ID:    2,
					Name:  "Notebook",
					Price: 1200,
					Stock: 10,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 2,
					Price:    500,
				},
				{
					ItemID:   2,
					Quantity: 1,
					Price:    1200,
				},
			},

			total: 2200,

			wantStatus:         "paid",
			wantPaymentSuccess: true,
			wantCreateCalled:   true,
			wantUpdateCalled:   true,
		},
		{
			name: "payment boundary at 4000 fails",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Name:  "Product",
					Price: 4000,
					Stock: 10,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
					Price:    4000,
				},
			},

			total: 4000,

			wantStatus:         "failed",
			wantPaymentSuccess: false,
			wantCreateCalled:   true,
			wantUpdateCalled:   true,
		},
		{
			name: "item does not exist",

			itemsInRepo: map[int]*models.Item{},

			lineItems: []models.LineItem{
				{
					ItemID:   999,
					Quantity: 1,
					Price:    1000,
				},
			},

			total: 1000,

			wantErr:          ErrItemNotFound,
			wantCreateCalled: false,
			wantUpdateCalled: false,
		},
		{
			name: "quantity exceeds stock",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Name:  "Laptop",
					Price: 3000,
					Stock: 2,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 3,
					Price:    3000,
				},
			},

			total: 9000,

			wantErr:          validation.ErrInsufficientStock,
			wantCreateCalled: false,
			wantUpdateCalled: false,
		},
		{
			name: "item repository fails",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Stock: 10,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
					Price:    1000,
				},
			},

			total: 1000,

			itemRepoErr:      errFakeRepository,
			wantErr:          errFakeRepository,
			wantCreateCalled: false,
			wantUpdateCalled: false,
		},
		{
			name: "create order repository fails",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Stock: 10,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
					Price:    1000,
				},
			},

			total: 1000,

			createRepoErr:    errFakeRepository,
			wantErr:          errFakeRepository,
			wantCreateCalled: true,
			wantUpdateCalled: false,
		},
		{
			name: "update order status repository fails",

			itemsInRepo: map[int]*models.Item{
				1: {
					ID:    1,
					Stock: 10,
				},
			},

			lineItems: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
					Price:    1000,
				},
			},

			total: 1000,

			updateRepoErr:    errFakeRepository,
			wantErr:          errFakeRepository,
			wantCreateCalled: true,
			wantUpdateCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			itemRepo := &fakeItemRepository{
				items: tt.itemsInRepo,
				err:   tt.itemRepoErr,
			}

			orderRepo := &fakeOrderRepository{
				createErr: tt.createRepoErr,
				updateErr: tt.updateRepoErr,
			}

			service := NewOrderService(
				orderRepo,
				itemRepo,
			)

			// Act
			got, err := service.CreateOrder(
				context.Background(),
				1,
				tt.lineItems,
				tt.total,
			)

			// Assert error contract
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"CreateOrder() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			// If an error was expected, there is no result to inspect.
			if tt.wantErr != nil {
				if got != nil {
					t.Errorf(
						"CreateOrder() result = %#v, want nil",
						got,
					)
				}

				if orderRepo.createCalled != tt.wantCreateCalled {
					t.Errorf(
						"CreateOrder called = %v, want %v",
						orderRepo.createCalled,
						tt.wantCreateCalled,
					)
				}

				if orderRepo.updateCalled != tt.wantUpdateCalled {
					t.Errorf(
						"UpdateOrderStatus called = %v, want %v",
						orderRepo.updateCalled,
						tt.wantUpdateCalled,
					)
				}

				return
			}

			if got == nil {
				t.Fatal("CreateOrder() returned nil result")
			}

			if got.Order == nil {
				t.Fatal("CreateOrder() returned nil order")
			}

			if got.Order.Status != tt.wantStatus {
				t.Errorf(
					"order status = %q, want %q",
					got.Order.Status,
					tt.wantStatus,
				)
			}

			if got.Payment.Success != tt.wantPaymentSuccess {
				t.Errorf(
					"payment success = %v, want %v",
					got.Payment.Success,
					tt.wantPaymentSuccess,
				)
			}

			if orderRepo.createCalled != tt.wantCreateCalled {
				t.Errorf(
					"CreateOrder repository called = %v, want %v",
					orderRepo.createCalled,
					tt.wantCreateCalled,
				)
			}

			if orderRepo.updateCalled != tt.wantUpdateCalled {
				t.Errorf(
					"UpdateOrderStatus repository called = %v, want %v",
					orderRepo.updateCalled,
					tt.wantUpdateCalled,
				)
			}

			if orderRepo.createdStatus != "pending" {
				t.Errorf(
					"initial order status = %q, want %q",
					orderRepo.createdStatus,
					"pending",
				)
			}

			if orderRepo.updatedStatus != tt.wantStatus {
				t.Errorf(
					"updated order status = %q, want %q",
					orderRepo.updatedStatus,
					tt.wantStatus,
				)
			}
		})
	}
}
