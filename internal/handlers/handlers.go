package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/models"
	"checkout-api/internal/services"

	"github.com/go-playground/validator/v10"
	httpSwagger "github.com/swaggo/http-swagger"
)

type AuthService interface {
	SaveUser(
		ctx context.Context,
		email string,
		password []byte,
	) error

	Authenticate(
		ctx context.Context,
		email string,
		password []byte,
	) (models.User, error)

	UserExists(
		ctx context.Context,
		id int,
	) (bool, error)

	GetUserByID(
		ctx context.Context,
		id int,
	) (models.User, error)

	GetUserInventory(
	ctx context.Context,
	userID int,
	) ([]models.InventoryItem, error)
}
type TokenService interface {
	CreateAccessToken(
		userID int,
	) (string, error)

	CreateRefreshToken(
		ctx context.Context,
		userID int,
	) (
		string,
		time.Time,
		error,
	)

	RotateRefreshToken(
		ctx context.Context,
		oldRefreshToken string,
	) (
		int,
		string,
		time.Time,
		error,
	)
	Logout(
		ctx context.Context,
		refreshToken string,
	) error
}

type ItemService interface {
	GetItem(ctx context.Context, id int) (*models.Item, error)
	GetItemsOffset(ctx context.Context, limit int, offset int) (services.ItemOffsetResult, error)
	GetItemsCursor(ctx context.Context, rawCursor string, limit int) (services.ItemCursorResult, error)
}

type CartService interface {
	GetUserCartOffset(ctx context.Context, userID int, limit int, offset int) (services.CartOffsetResult, error)
	GetUserCartCursor(ctx context.Context, userID int, rawCursor string, limit int) (services.CartCursorResult, error)
	UpsertCartItem(ctx context.Context, userID int, itemID int, quantity int) error
	RemoveCartItem(ctx context.Context, userID int, itemID int) error
}

type OrderService interface {
	CreateOrder(ctx context.Context, userID int) (*services.CreateOrderResult, error)
	GetOrdersByUserIDOffset(ctx context.Context, userID int, limit int, offset int) (services.OrderOffsetResult, error)
	GetOrdersByUserIDCursor(ctx context.Context, userID int, rawCursor string, limit int) (services.OrderCursorResult, error)
	ProcessXsollaPayment(ctx context.Context, orderID int, transactionID int64) error
	UpdateOrderStatus(ctx context.Context, orderID int, status string) error
}
type XsollaService interface {
	CreatePayment(ctx context.Context, userID int, orderID int, items []services.XsollaItem) (*services.XsollaPaymentResponse, error)
	VerifyXsollaSignature(rawBody []byte, authHeader string) bool
}
type Handler struct {
	authService      AuthService
	itemService      ItemService
	cartService      CartService
	orderService     OrderService
	tokenService     TokenService
	xsollaService    XsollaService
	jwtSecret        []byte
	validate         *validator.Validate
	idempotencyCache map[string]*IdempotencyRecord
}

type IdempotencyRecord struct {
	Response   []byte
	StatusCode int
	Expiry     time.Time
}

func NewHandler(
	authService AuthService,
	itemService ItemService,
	cartService CartService,
	orderService OrderService,
	tokenService TokenService,
	xsollaService XsollaService,
	jwtSecret []byte,

) *Handler {
	return &Handler{
		authService:      authService,
		itemService:      itemService,
		cartService:      cartService,
		orderService:     orderService,
		tokenService:     tokenService,
		xsollaService:    xsollaService,
		jwtSecret:        jwtSecret,
		validate:         validator.New(),
		idempotencyCache: make(map[string]*IdempotencyRecord),
	}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)

	mux.Handle(
		"GET /swagger/",
		httpSwagger.WrapHandler,
	)
	mux.Handle(
		"GET /user/me",
		middlewares.AuthMiddleware(
			http.HandlerFunc(h.GetCurrentUser),
			h.jwtSecret,
		),
	)
	mux.Handle(
	"GET /user/inventory",
	middlewares.AuthMiddleware(
		http.HandlerFunc(h.GetUserInventory),
		h.jwtSecret,
	),
)
	mux.Handle(
		"GET /user/cart",
		middlewares.AuthMiddleware(http.HandlerFunc(h.GetUserCart), h.jwtSecret),
	)
	mux.Handle(
		"GET /user/cart/cursor",
		middlewares.AuthMiddleware(http.HandlerFunc(h.GetUserCartCursor), h.jwtSecret),
	)
	mux.Handle(
		"PATCH /user/cart/items/{item_id}",
		middlewares.AuthMiddleware(http.HandlerFunc(h.UpsertCartItem), h.jwtSecret),
	)
	mux.Handle(
		"DELETE /user/cart/items/{item_id}",
		middlewares.AuthMiddleware(http.HandlerFunc(h.RemoveCartItem), h.jwtSecret),
	)

	mux.Handle(
		"POST /orders",
		middlewares.AuthMiddleware(http.HandlerFunc(h.CreateOrder), h.jwtSecret),
	)
	mux.Handle(
		"GET /user/orders",
		middlewares.AuthMiddleware(http.HandlerFunc(h.GetUserOrders), h.jwtSecret),
	)
	mux.Handle(
		"GET /user/orders/cursor",
		middlewares.AuthMiddleware(http.HandlerFunc(h.GetUserOrdersCursor), h.jwtSecret),
	)
	mux.HandleFunc(
		"POST /webhook",
		h.XsollaWebhook,
	)

	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("GET /items/cursor", h.GetItemsCursor)
	mux.HandleFunc("GET /items/{item_id}", h.GetItemByID)

	mux.HandleFunc("POST /signup", h.CreateUser)
	mux.HandleFunc("POST /login", h.LoginUser)
	mux.HandleFunc("POST /token", h.IssueJWT)
	mux.HandleFunc("POST /logout", h.Logout)
	return mux
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func intQuery(r *http.Request, name string, def int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def, nil
	}

	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s cannot be negative", name)
	}

	return n, nil
}
