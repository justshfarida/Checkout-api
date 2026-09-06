package services

import (
	"context"
	"errors"

	"checkout-api/internal/models"
	"checkout-api/internal/repositories"
)

// Shared fake error used to simulate dependency failures.
var errFakeDependency = errors.New("fake dependency error")

// ======================================================
// FAKE PAYMENT PROCESSOR
// ======================================================

type fakePaymentProcessor struct {
	result PaymentResult
	err    error

	called bool
	amount int
}

func (f *fakePaymentProcessor) ProcessPayment(
	amount int,
) (PaymentResult, error) {
	f.called = true
	f.amount = amount

	if f.err != nil {
		return PaymentResult{}, f.err
	}

	return f.result, nil
}

// Compile-time check that the fake satisfies the interface.
var _ PaymentProcessor = (*fakePaymentProcessor)(nil)

// ======================================================
// FAKE ITEM REPOSITORY
// ======================================================

type fakeItemRepository struct {
	// GetItem
	items map[int]*models.Item
	err   error

	// GetItemsOffset
	offsetItems []*models.Item
	offsetErr   error

	offsetLimit int
	offsetValue int

	// GetItemsCursor
	cursorItems []*models.Item
	cursorErr   error

	cursorLastID int
	cursorLimit  int

	// CountItems
	count    int
	countErr error
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
	f.offsetLimit = limit
	f.offsetValue = offset

	if f.offsetErr != nil {
		return nil, f.offsetErr
	}

	return f.offsetItems, nil
}

func (f *fakeItemRepository) GetItemsCursor(
	ctx context.Context,
	lastID int,
	limit int,
) ([]*models.Item, error) {
	f.cursorLastID = lastID
	f.cursorLimit = limit

	if f.cursorErr != nil {
		return nil, f.cursorErr
	}

	return f.cursorItems, nil
}

func (f *fakeItemRepository) CountItems(
	ctx context.Context,
) (int, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}

	return f.count, nil
}

var _ repositories.ItemRepository = (*fakeItemRepository)(nil)

// ======================================================
// FAKE CART REPOSITORY
// ======================================================

type fakeCartRepository struct {
	// GetUserCartOffset
	items  []models.CartItem
	getErr error

	// GetUserCartCursor
	cursorItems []models.CartItem
	cursorErr   error

	// GetUserCartTotal
	total    int
	totalErr error

	// DeleteUserCart
	deleteErr     error
	deleteCalled  bool
	deletedUserID int

	// UpsertCartItem
	upsertErr      error
	upsertCalled   bool
	upsertUserID   int
	upsertItemID   int
	upsertQuantity int

	// RemoveCartItem
	removeErr    error
	removeCalled bool
	removeUserID int
	removeItemID int

	// Captured pagination args
	offsetLimit  int
	offsetValue  int
	cursorLastID int
	cursorLimit  int
}

func (f *fakeCartRepository) GetUserCartOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.CartItem, error) {
	f.offsetLimit = limit
	f.offsetValue = offset

	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.items, nil
}

func (f *fakeCartRepository) DeleteUserCart(
	ctx context.Context,
	userID int,
) error {
	f.deleteCalled = true
	f.deletedUserID = userID

	return f.deleteErr
}
func (f *fakeCartRepository) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	f.upsertCalled = true
	f.upsertUserID = userID
	f.upsertItemID = itemID
	f.upsertQuantity = quantity

	return f.upsertErr
}

func (f *fakeCartRepository) GetUserCartCursor(
	ctx context.Context,
	userID int,
	lastID int,
	limit int,
) ([]models.CartItem, error) {
	f.cursorLastID = lastID
	f.cursorLimit = limit

	if f.cursorErr != nil {
		return nil, f.cursorErr
	}

	return f.cursorItems, nil
}

func (f *fakeCartRepository) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	f.removeCalled = true
	f.removeUserID = userID
	f.removeItemID = itemID

	return f.removeErr
}
func (f *fakeCartRepository) GetUserCartTotal(
	ctx context.Context,
	userID int,
) (int, error) {
	if f.totalErr != nil {
		return 0, f.totalErr
	}

	return f.total, nil
}

var _ repositories.CartRepository = (*fakeCartRepository)(nil)

// ======================================================
// FAKE ORDER REPOSITORY
// ======================================================

type fakeOrderRepository struct {
	createErr   error
	updateErr   error
	markPaidErr error

	createCalled   bool
	updateCalled   bool
	markPaidCalled bool

	createdTotal  int
	createdStatus string
	createdItems  []models.LineItem

	updatedOrderID int
	updatedStatus  string
}

func (f *fakeOrderRepository) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	f.createCalled = true
	f.createdTotal = total
	f.createdStatus = status
	f.createdItems = items

	if f.createErr != nil {
		return nil, f.createErr
	}

	return &models.Order{
		ID:     10,
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
	f.updatedOrderID = orderID
	f.updatedStatus = status

	return f.updateErr
}

func (f *fakeOrderRepository) MarkOrderPaid(
	ctx context.Context,
	orderID int,
	items []models.LineItem,
) error {
	f.markPaidCalled = true

	return f.markPaidErr
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

var _ repositories.OrderRepository = (*fakeOrderRepository)(nil)
