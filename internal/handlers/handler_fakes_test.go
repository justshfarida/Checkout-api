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
			Subject: strconv.Itoa(userID),
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(time.Hour),
			),
		},
	)

	signed, err := token.SignedString(
		testJWTSecret,
	)
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
	token TokenService,
) *Handler {
	return newTestHandlerWithXsolla(auth, item, cart, order, token, nil)
}

func newTestHandlerWithXsolla(
	auth AuthService,
	item ItemService,
	cart CartService,
	order OrderService,
	token TokenService,
	xsolla XsollaService,
) *Handler {
	return NewHandler(
		auth,
		item,
		cart,
		order,
		token,
		xsolla,
		testJWTSecret,
	)
}

// -------------------------------------------------------
// AUTH SERVICE FAKE
// -------------------------------------------------------

type fakeAuthService struct {
	authenticateUser models.User
	authenticateErr  error

	saveUserErr error

	authenticateCalled bool
	saveUserCalled     bool

	authenticateEmail    string
	authenticatePassword []byte

	savedEmail    string
	savedPassword []byte

	// UserExists
	userExists       bool
	userExistsErr    error
	userExistsCalled bool
	userExistsID     int

	// GetUserByID
	userByID       models.User
	userByIDErr    error
	getUserByIDArg int

	// GetUserInventory
	inventory       []models.InventoryItem
	inventoryErr    error
	inventoryUserID int
}

func (f *fakeAuthService) Authenticate(
	ctx context.Context,
	email string,
	password []byte,
) (models.User, error) {
	f.authenticateCalled = true
	f.authenticateEmail = email
	f.authenticatePassword = append(
		[]byte(nil),
		password...,
	)

	if f.authenticateErr != nil {
		return models.User{},
			f.authenticateErr
	}

	return f.authenticateUser,
		nil
}

func (f *fakeAuthService) SaveUser(
	ctx context.Context,
	email string,
	password []byte,
) error {
	f.saveUserCalled = true
	f.savedEmail = email
	f.savedPassword = append(
		[]byte(nil),
		password...,
	)

	return f.saveUserErr
}

func (f *fakeAuthService) UserExists(
	ctx context.Context,
	id int,
) (bool, error) {
	f.userExistsCalled = true
	f.userExistsID = id

	return f.userExists, f.userExistsErr
}

func (f *fakeAuthService) GetUserByID(
	ctx context.Context,
	id int,
) (models.User, error) {
	f.getUserByIDArg = id

	return f.userByID, f.userByIDErr
}

func (f *fakeAuthService) GetUserInventory(
	ctx context.Context,
	userID int,
) ([]models.InventoryItem, error) {
	f.inventoryUserID = userID

	return f.inventory, f.inventoryErr
}

var _ AuthService = (*fakeAuthService)(nil)

// -------------------------------------------------------
// TOKEN SERVICE FAKE
// -------------------------------------------------------

type fakeTokenService struct {
	accessToken    string
	accessTokenErr error

	refreshToken     string
	refreshExpiresAt time.Time
	refreshTokenErr  error

	rotateUserID    int
	rotatedToken    string
	rotateExpiresAt time.Time
	rotateErr       error

	createAccessCalled  bool
	createRefreshCalled bool
	rotateCalled        bool

	accessUserID  int
	refreshUserID int

	oldRefreshToken string

	// Logout
	logoutErr    error
	logoutCalled bool
	logoutToken  string
}

func (f *fakeTokenService) CreateAccessToken(
	userID int,
) (string, error) {
	f.createAccessCalled = true
	f.accessUserID = userID

	if f.accessTokenErr != nil {
		return "",
			f.accessTokenErr
	}

	return f.accessToken,
		nil
}

func (f *fakeTokenService) CreateRefreshToken(
	ctx context.Context,
	userID int,
) (
	string,
	time.Time,
	error,
) {
	f.createRefreshCalled = true
	f.refreshUserID = userID

	if f.refreshTokenErr != nil {
		return "",
			time.Time{},
			f.refreshTokenErr
	}

	return f.refreshToken,
		f.refreshExpiresAt,
		nil
}

func (f *fakeTokenService) RotateRefreshToken(
	ctx context.Context,
	oldRefreshToken string,
) (
	int,
	string,
	time.Time,
	error,
) {
	f.rotateCalled = true
	f.oldRefreshToken = oldRefreshToken

	if f.rotateErr != nil {
		return 0,
			"",
			time.Time{},
			f.rotateErr
	}

	return f.rotateUserID,
		f.rotatedToken,
		f.rotateExpiresAt,
		nil
}

func (f *fakeTokenService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	f.logoutCalled = true
	f.logoutToken = refreshToken

	return f.logoutErr
}

var _ TokenService = (*fakeTokenService)(nil)

// -------------------------------------------------------
// ITEM SERVICE FAKE
// -------------------------------------------------------

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

func (f *fakeItemService) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	f.getCalled = true

	return f.item,
		f.getItemErr
}

func (f *fakeItemService) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) (
	services.ItemOffsetResult,
	error,
) {
	f.offsetCalled = true

	return f.offsetResult,
		f.offsetErr
}

func (f *fakeItemService) GetItemsCursor(
	ctx context.Context,
	rawCursor string,
	limit int,
) (
	services.ItemCursorResult,
	error,
) {
	f.cursorCalled = true

	return f.cursorResult,
		f.cursorErr
}

var _ ItemService = (*fakeItemService)(nil)

// -------------------------------------------------------
// CART SERVICE FAKE
// -------------------------------------------------------

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
) (
	services.CartOffsetResult,
	error,
) {
	f.offsetCalled = true
	f.userID = userID

	return f.offsetResult,
		f.offsetErr
}

func (f *fakeCartService) GetUserCartCursor(
	ctx context.Context,
	userID int,
	rawCursor string,
	limit int,
) (
	services.CartCursorResult,
	error,
) {
	f.cursorCalled = true
	f.userID = userID

	return f.cursorResult,
		f.cursorErr
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

// -------------------------------------------------------
// ORDER SERVICE FAKE
// -------------------------------------------------------

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

	// ProcessXsollaPayment
	processErr           error
	processCalled        bool
	processOrderID       int
	processTransactionID int64

	// UpdateOrderStatus
	updateErr     error
	updateCalled  bool
	updateOrderID int
	updateStatus  string
}

func (f *fakeOrderService) CreateOrder(
	ctx context.Context,
	userID int,
) (
	*services.CreateOrderResult,
	error,
) {
	f.createCalls++
	f.userID = userID

	return f.createResult,
		f.createErr
}

func (f *fakeOrderService) GetOrdersByUserIDOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) (
	services.OrderOffsetResult,
	error,
) {
	f.offsetCalled = true
	f.userID = userID

	return f.offsetResult,
		f.offsetErr
}

func (f *fakeOrderService) GetOrdersByUserIDCursor(
	ctx context.Context,
	userID int,
	rawCursor string,
	limit int,
) (
	services.OrderCursorResult,
	error,
) {
	f.cursorCalled = true
	f.userID = userID

	return f.cursorResult,
		f.cursorErr
}

func (f *fakeOrderService) ProcessXsollaPayment(
	ctx context.Context,
	orderID int,
	transactionID int64,
) error {
	f.processCalled = true
	f.processOrderID = orderID
	f.processTransactionID = transactionID

	return f.processErr
}

func (f *fakeOrderService) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	f.updateCalled = true
	f.updateOrderID = orderID
	f.updateStatus = status

	return f.updateErr
}

var _ OrderService = (*fakeOrderService)(nil)

// -------------------------------------------------------
// XSOLLA SERVICE FAKE
// -------------------------------------------------------

type fakeXsollaService struct {
	signatureValid bool

	verifyCalled bool
	verifiedBody []byte
	verifiedAuth string
}

func (f *fakeXsollaService) CreatePayment(
	ctx context.Context,
	userID int,
	orderID int,
	items []services.XsollaItem,
) (*services.XsollaPaymentResponse, error) {
	return nil, nil
}

func (f *fakeXsollaService) VerifyXsollaSignature(
	rawBody []byte,
	authHeader string,
) bool {
	f.verifyCalled = true
	f.verifiedBody = append([]byte(nil), rawBody...)
	f.verifiedAuth = authHeader

	return f.signatureValid
}

var _ XsollaService = (*fakeXsollaService)(nil)
