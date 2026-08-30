package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/services"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret")

// -----------------------------------------------------------------------------
// Fake Repository
// -----------------------------------------------------------------------------

type fakeRepository struct {
	items     []*models.Item
	cartItems []models.CartItem
	orders    []models.UserOrder
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		items: []*models.Item{
			{
				ID:          1,
				Name:        "Laptop",
				Description: "Test laptop",
				Price:       1200,
				Stock:       10,
				CreatedAt:   time.Now(),
			},
			{
				ID:          2,
				Name:        "Mouse",
				Description: "Test mouse",
				Price:       500,
				Stock:       20,
				CreatedAt:   time.Now(),
			},
		},
		cartItems: []models.CartItem{},
		orders:    []models.UserOrder{},
	}
}

// -----------------------------------------------------------------------------
// ItemRepository
// -----------------------------------------------------------------------------

func (r *fakeRepository) GetItem(
	ctx context.Context,
	id int,
) (*models.Item, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}

	return nil, nil
}

func (r *fakeRepository) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) ([]*models.Item, error) {
	if offset >= len(r.items) {
		return []*models.Item{}, nil
	}

	end := offset + limit

	if end > len(r.items) {
		end = len(r.items)
	}

	return r.items[offset:end], nil
}

func (r *fakeRepository) GetItemsCursor(
	ctx context.Context,
	lastID int,
	limit int,
) ([]*models.Item, error) {
	result := make([]*models.Item, 0)

	for _, item := range r.items {
		if item.ID > lastID {
			result = append(
				result,
				item,
			)
		}

		if len(result) == limit {
			break
		}
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// CartRepository
// -----------------------------------------------------------------------------

func (r *fakeRepository) UpsertCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	item, _ := r.GetItem(
		ctx,
		itemID,
	)

	if item == nil {
		return services.ErrItemNotFound
	}

	for i := range r.cartItems {
		if r.cartItems[i].ID == itemID {
			r.cartItems[i].Quantity = quantity
			return nil
		}
	}

	r.cartItems = append(
		r.cartItems,
		models.CartItem{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Stock:       item.Stock,
			CreatedAt:   item.CreatedAt,
			Quantity:    quantity,
		},
	)

	return nil
}

func (r *fakeRepository) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) error {
	for i, item := range r.cartItems {
		if item.ID == itemID {
			r.cartItems = append(
				r.cartItems[:i],
				r.cartItems[i+1:]...,
			)

			return nil
		}
	}

	return nil
}

func (r *fakeRepository) DeleteUserCart(
	ctx context.Context,
	userID int,
) error {
	r.cartItems = nil
	return nil
}

func (r *fakeRepository) GetUserCartOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.CartItem, error) {
	if offset >= len(r.cartItems) {
		return []models.CartItem{}, nil
	}

	end := offset + limit

	if end > len(r.cartItems) {
		end = len(r.cartItems)
	}

	return r.cartItems[offset:end], nil
}

func (r *fakeRepository) GetUserCartCursor(
	ctx context.Context,
	userID int,
	lastID int,
	limit int,
) ([]models.CartItem, error) {
	result := make([]models.CartItem, 0)

	for _, item := range r.cartItems {
		if item.ID > lastID {
			result = append(
				result,
				item,
			)
		}

		if len(result) == limit {
			break
		}
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// OrderRepository
// -----------------------------------------------------------------------------

func (r *fakeRepository) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	order := &models.Order{
		ID:     len(r.orders) + 1,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}

	userOrder := models.UserOrder{
		ID:     order.ID,
		UserID: userID,
		Total:  total,
		Status: status,
		Items:  []models.OrderItem{},
	}

	for _, line := range items {
		item, _ := r.GetItem(
			ctx,
			line.ItemID,
		)

		if item == nil {
			return nil, services.ErrItemNotFound
		}

		userOrder.Items = append(
			userOrder.Items,
			models.OrderItem{
				ItemID:   item.ID,
				Name:     item.Name,
				Price:    line.Price,
				Quantity: line.Quantity,
			},
		)
	}

	r.orders = append(
		r.orders,
		userOrder,
	)

	return order, nil
}

func (r *fakeRepository) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	for i := range r.orders {
		if r.orders[i].ID == orderID {
			r.orders[i].Status = status
			return nil
		}
	}

	return nil
}

func (r *fakeRepository) GetOrdersByUserIDOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.UserOrder, error) {
	filtered := make(
		[]models.UserOrder,
		0,
	)

	for _, order := range r.orders {
		if order.UserID == userID {
			filtered = append(
				filtered,
				order,
			)
		}
	}

	if offset >= len(filtered) {
		return []models.UserOrder{}, nil
	}

	end := offset + limit

	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end], nil
}

func (r *fakeRepository) GetOrdersByUserIDCursor(
	ctx context.Context,
	userID int,
	lastID int,
	limit int,
) ([]models.UserOrder, error) {
	result := make(
		[]models.UserOrder,
		0,
	)

	for _, order := range r.orders {
		if order.UserID != userID {
			continue
		}

		if lastID == 0 || order.ID < lastID {
			result = append(
				result,
				order,
			)
		}

		if len(result) == limit {
			break
		}
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// UserRepository
// -----------------------------------------------------------------------------
//
// These methods are required because the same fake repository is also supplied
// to AuthService. The tests below don't exercise authentication persistence.

func (r *fakeRepository) FindUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	return models.User{}, nil
}

func (r *fakeRepository) SaveUser(
	ctx context.Context,
	email string,
	hash []byte,
) error {
	return nil
}

func (r *fakeRepository) SaveRefreshToken(
	ctx context.Context,
	userID int,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	return nil
}

func (r *fakeRepository) RotateRefreshToken(
	ctx context.Context,
	oldTokenHash []byte,
	newTokenHash []byte,
	newExpiresAt time.Time,
) (int, error) {
	return 1, nil
}

// -----------------------------------------------------------------------------
// Test Setup
// -----------------------------------------------------------------------------

func newTestHandler(
	t *testing.T,
) *Handler {
	t.Helper()

	repo := newFakeRepository()

	authService :=
		services.NewAuthService(
			repo,
		)

	itemService :=
		services.NewItemService(
			repo,
		)

	cartService :=
		services.NewCartService(
			repo,
			repo,
		)

	orderService :=
		services.NewOrderService(
			repo,
			repo,
		)

	return NewHandler(
		authService,
		itemService,
		cartService,
		orderService,
		testSecret,
	)
}

func testJWT(
	t *testing.T,
	userID string,
) string {
	t.Helper()

	token :=
		jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.RegisteredClaims{
				Subject: userID,
				ExpiresAt: jwt.NewNumericDate(
					time.Now().
						Add(time.Hour),
				),
			},
		)

	signed, err :=
		token.SignedString(
			testSecret,
		)

	if err != nil {
		t.Fatalf(
			"failed to sign test jwt: %v",
			err,
		)
	}

	return signed
}

func authenticatedRequest(
	t *testing.T,
	method string,
	path string,
	body string,
) *http.Request {
	t.Helper()

	var req *http.Request

	if body == "" {
		req =
			httptest.NewRequest(
				method,
				path,
				nil,
			)
	} else {
		req =
			httptest.NewRequest(
				method,
				path,
				strings.NewReader(body),
			)

		req.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+testJWT(t, "1"),
	)

	return req
}

// -----------------------------------------------------------------------------
// ITEMS
// -----------------------------------------------------------------------------

func TestGetItems(
	t *testing.T,
) {
	h := newTestHandler(t)
	router := h.Routes()

	req :=
		httptest.NewRequest(
			http.MethodGet,
			"/items?limit=10&offset=0",
			nil,
		)

	w := httptest.NewRecorder()

	router.ServeHTTP(
		w,
		req,
	)

	if w.Code != http.StatusOK {
		t.Errorf(
			"status = %d, want %d",
			w.Code,
			http.StatusOK,
		)
	}

	if !strings.Contains(
		w.Body.String(),
		"Laptop",
	) {
		t.Errorf(
			"body missing Laptop, got: %s",
			w.Body.String(),
		)
	}
}

func TestGetItemsInvalidLimit(
	t *testing.T,
) {
	h := newTestHandler(t)
	router := h.Routes()

	req :=
		httptest.NewRequest(
			http.MethodGet,
			"/items?limit=hello",
			nil,
		)

	w := httptest.NewRecorder()

	router.ServeHTTP(
		w,
		req,
	)

	if w.Code != http.StatusBadRequest {
		t.Errorf(
			"status = %d, want %d",
			w.Code,
			http.StatusBadRequest,
		)
	}

	if !strings.Contains(
		w.Body.String(),
		"INVALID_QUERY_PARAMETER",
	) {
		t.Errorf(
			"expected unified error envelope, got: %s",
			w.Body.String(),
		)
	}
}

func TestGetItemByID(
	t *testing.T,
) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid item",
			path:       "/items/1",
			wantStatus: http.StatusOK,
			wantBody:   "Laptop",
		},
		{
			name:       "item not found",
			path:       "/items/999",
			wantStatus: http.StatusNotFound,
			wantBody:   "ITEM_NOT_FOUND",
		},
		{
			name:       "invalid ID",
			path:       "/items/abc",
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_ITEM_ID",
		},
		{
			name:       "negative ID",
			path:       "/items/-1",
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_ITEM_ID",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				h := newTestHandler(t)
				router := h.Routes()

				req :=
					httptest.NewRequest(
						http.MethodGet,
						tt.path,
						nil,
					)

				w :=
					httptest.NewRecorder()

				router.ServeHTTP(
					w,
					req,
				)

				if w.Code != tt.wantStatus {
					t.Errorf(
						"status = %d, want %d",
						w.Code,
						tt.wantStatus,
					)
				}

				if tt.wantBody != "" &&
					!strings.Contains(
						w.Body.String(),
						tt.wantBody,
					) {

					t.Errorf(
						"body missing %q, got: %s",
						tt.wantBody,
						w.Body.String(),
					)
				}
			},
		)
	}
}

// -----------------------------------------------------------------------------
// CART
// -----------------------------------------------------------------------------

func TestGetUserCartUnauthorized(
	t *testing.T,
) {
	h := newTestHandler(t)
	router := h.Routes()

	req :=
		httptest.NewRequest(
			http.MethodGet,
			"/user/cart",
			nil,
		)

	w := httptest.NewRecorder()

	router.ServeHTTP(
		w,
		req,
	)

	if w.Code != http.StatusUnauthorized {
		t.Errorf(
			"status = %d, want %d",
			w.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestGetUserCartEmpty(
	t *testing.T,
) {
	h := newTestHandler(t)
	router := h.Routes()

	req :=
		authenticatedRequest(
			t,
			http.MethodGet,
			"/user/cart",
			"",
		)

	w := httptest.NewRecorder()

	router.ServeHTTP(
		w,
		req,
	)

	if w.Code != http.StatusOK {
		t.Errorf(
			"status = %d, want %d",
			w.Code,
			http.StatusOK,
		)
	}

	if !strings.Contains(
		w.Body.String(),
		`"items":[]`,
	) {
		t.Errorf(
			"expected empty cart, got: %s",
			w.Body.String(),
		)
	}
}

func TestUpsertAndGetUserCart(
	t *testing.T,
) {
	h := newTestHandler(t)
	router := h.Routes()

	addReq :=
		authenticatedRequest(
			t,
			http.MethodPatch,
			"/user/cart/items/1",
			`{"quantity":2}`,
		)

	addW :=
		httptest.NewRecorder()

	router.ServeHTTP(
		addW,
		addReq,
	)

	if addW.Code != http.StatusNoContent {
		t.Fatalf(
			"upsert status = %d, want %d; body=%s",
			addW.Code,
			http.StatusNoContent,
			addW.Body.String(),
		)
	}

	getReq :=
		authenticatedRequest(
			t,
			http.MethodGet,
			"/user/cart",
			"",
		)

	getW :=
		httptest.NewRecorder()

	router.ServeHTTP(
		getW,
		getReq,
	)

	if getW.Code != http.StatusOK {
		t.Fatalf(
			"get cart status = %d, want %d",
			getW.Code,
			http.StatusOK,
		)
	}

	if !strings.Contains(
		getW.Body.String(),
		`"name":"Laptop"`,
	) {
		t.Errorf(
			"cart body missing Laptop: %s",
			getW.Body.String(),
		)
	}

	if !strings.Contains(
		getW.Body.String(),
		`"quantity":2`,
	) {
		t.Errorf(
			"cart body missing quantity 2: %s",
			getW.Body.String(),
		)
	}
}

func TestUpsertCartItemValidation(
	t *testing.T,
) {
	h := newTestHandler(t)
	router := h.Routes()

	req :=
		authenticatedRequest(
			t,
			http.MethodPatch,
			"/user/cart/items/1",
			`{"quantity":0}`,
		)

	w :=
		httptest.NewRecorder()

	router.ServeHTTP(
		w,
		req,
	)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf(
			"status = %d, want %d",
			w.Code,
			http.StatusUnprocessableEntity,
		)
	}

	if !strings.Contains(
		w.Body.String(),
		"VALIDATION_FAILED",
	) {
		t.Errorf(
			"expected validation error envelope, got: %s",
			w.Body.String(),
		)
	}
}

// -----------------------------------------------------------------------------
// ORDERS
// -----------------------------------------------------------------------------

func TestCreateOrder(
	t *testing.T,
) {
	tests := []struct {
		name       string
		auth       bool
		idempotKey string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid order",
			auth:       true,
			idempotKey: "key-001",
			body:       `{"line_items":[{"item_id":1,"quantity":1,"price":1200}],"total":1200}`,
			wantStatus: http.StatusCreated,
			wantBody:   `"status":"paid"`,
		},
		{
			name:       "empty items",
			auth:       true,
			idempotKey: "key-002",
			body:       `{"line_items":[],"total":1200}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "VALIDATION_FAILED",
		},
		{
			name:       "missing authorization",
			auth:       false,
			idempotKey: "key-003",
			body:       `{"line_items":[{"item_id":1,"quantity":1,"price":1200}],"total":1200}`,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "UNAUTHORIZED",
		},
		{
			name:       "missing idempotency key",
			auth:       true,
			idempotKey: "",
			body:       `{"line_items":[{"item_id":1,"quantity":1,"price":1200}],"total":1200}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "IDEMPOTENCY_KEY_REQUIRED",
		},
		{
			name:       "invalid body",
			auth:       true,
			idempotKey: "key-004",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "INVALID_JSON",
		},
		{
			name:       "insufficient stock",
			auth:       true,
			idempotKey: "key-005",
			body:       `{"line_items":[{"item_id":1,"quantity":100,"price":1200}],"total":120000}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "INSUFFICIENT_STOCK",
		},
	}

	for _, tt := range tests {

		t.Run(
			tt.name,
			func(t *testing.T) {
				h :=
					newTestHandler(t)

				router :=
					h.Routes()

				req :=
					httptest.NewRequest(
						http.MethodPost,
						"/orders",
						strings.NewReader(
							tt.body,
						),
					)

				req.Header.Set(
					"Content-Type",
					"application/json",
				)

				if tt.auth {
					req.Header.Set(
						"Authorization",
						"Bearer "+testJWT(
							t,
							"1",
						),
					)
				}

				if tt.idempotKey != "" {
					req.Header.Set(
						"Idempotency-Key",
						tt.idempotKey,
					)
				}

				w :=
					httptest.NewRecorder()

				router.ServeHTTP(
					w,
					req,
				)

				if w.Code !=
					tt.wantStatus {

					t.Errorf(
						"status = %d, want %d; body=%s",
						w.Code,
						tt.wantStatus,
						w.Body.String(),
					)
				}

				if tt.wantBody != "" &&
					!strings.Contains(
						w.Body.String(),
						tt.wantBody,
					) {

					t.Errorf(
						"body missing %q, got: %s",
						tt.wantBody,
						w.Body.String(),
					)
				}
			},
		)
	}
}
