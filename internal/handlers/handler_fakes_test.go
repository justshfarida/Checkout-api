package handlers

import (
	"context"
	"strconv"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/services"

	"github.com/golang-jwt/jwt/v5"
)

var testJWTSecret = []byte("test-secret-key")

func testAccessToken(userID int) string {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	)

	signed, err := token.SignedString(testJWTSecret)
	if err != nil {
		panic(err)
	}
	return signed
}

func newTestHandler(
	auth AuthService,
	item ItemService,
	cart CartService,
	order OrderService,
) *Handler {
	return NewHandler(auth, item, cart, order, testJWTSecret)
}

type fakeAuthService struct {
	user    models.User
	findErr error

	saveUserErr error

	saveRefreshErr error

	rotateUserID int
	rotateErr    error

	findCalled        bool
	saveUserCalled    bool
	saveRefreshCalled bool
	rotateCalled      bool

	foundEmail string
	savedEmail string
	savedHash  []byte
}

func (f *fakeAuthService) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	f.findCalled = true
	f.foundEmail = email
	return f.user, f.findErr
}

func (f *fakeAuthService) SaveUser(ctx context.Context, email string, hash []byte) error {
	f.saveUserCalled = true
	f.savedEmail = email
	f.savedHash = append([]byte(nil), hash...)
	return f.saveUserErr
}

func (f *fakeAuthService) SaveRefreshToken(
	ctx context.Context,
	userID int,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	f.saveRefreshCalled = true
	return f.saveRefreshErr
}

func (f *fakeAuthService) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash []byte,
	newTokenHash []byte,
	newExpiresAt time.Time,
) (int, error) {
	f.rotateCalled = true
	return f.rotateUserID, f.rotateErr
}

var _ AuthService = (*fakeAuthService)(nil)

type fakeItemService struct {
	item       *models.Item
	getItemErr error

	offsetResult services.ItemOffsetResult
	offsetErr    error

	cursorResult services.ItemCursorResult
	cursorErr    error

	getCalled    bool
	offsetCalled bool
	cursorCalled bool
}

func (f *fakeItemService) GetItem(ctx context.Context, id int) (*models.Item, error) {
	f.getCalled = true
	return f.item, f.getItemErr
}

func (f *fakeItemService) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) (services.ItemOffsetResult, error) {
	f.offsetCalled = true
	return f.offsetResult, f.offsetErr
}

func (f *fakeItemService) GetItemsCursor(
	ctx context.Context,
	rawCursor string,
	limit int,
) (services.ItemCursorResult, error) {
	f.cursorCalled = true
	return f.cursorResult, f.cursorErr
}

var _ ItemService = (*fakeItemService)(nil)

type fakeCartService struct {
	offsetResult services.CartOffsetResult
	offsetErr    error

	cursorResult services.CartCursorResult
	cursorErr    error

	upsertErr error
	removeErr error

	upsertCalled bool
	removeCalled bool
	offsetCalled bool
	cursorCalled bool

	userID   int
	itemID   int
	quantity int
}

func (f *fakeCartService) GetUserCartOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) (services.CartOffsetResult, error) {
	f.offsetCalled = true
	f.userID = userID
	return f.offsetResult, f.offsetErr
}

func (f *fakeCartService) GetUserCartCursor(
	ctx context.Context,
	userID int,
	rawCursor string,
	limit int,
) (services.CartCursorResult, error) {
	f.cursorCalled = true
	f.userID = userID
	return f.cursorResult, f.cursorErr
}

func (f *fakeCartService) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	f.upsertCalled = true
	f.userID = userID
	f.itemID = itemID
	f.quantity = quantity
	return f.upsertErr
}

func (f *fakeCartService) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	f.removeCalled = true
	f.userID = userID
	f.itemID = itemID
	return f.removeErr
}

var _ CartService = (*fakeCartService)(nil)

type fakeOrderService struct {
	createResult *services.CreateOrderResult
	createErr    error

	offsetResult services.OrderOffsetResult
	offsetErr    error

	cursorResult services.OrderCursorResult
	cursorErr    error

	createCalls  int
	offsetCalled bool
	cursorCalled bool
	userID       int
}

func (f *fakeOrderService) CreateOrder(
	ctx context.Context,
	userID int,
) (*services.CreateOrderResult, error) {
	f.createCalls++
	f.userID = userID
	return f.createResult, f.createErr
}

func (f *fakeOrderService) GetOrdersByUserIDOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) (services.OrderOffsetResult, error) {
	f.offsetCalled = true
	f.userID = userID
	return f.offsetResult, f.offsetErr
}

func (f *fakeOrderService) GetOrdersByUserIDCursor(
	ctx context.Context,
	userID int,
	rawCursor string,
	limit int,
) (services.OrderCursorResult, error) {
	f.cursorCalled = true
	f.userID = userID
	return f.cursorResult, f.cursorErr
}

var _ OrderService = (*fakeOrderService)(nil)
