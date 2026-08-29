package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/services"
	"strconv"
)

type Handler struct {
	authService  *services.AuthService
	itemService  *services.ItemService
	cartService  *services.CartService
	orderService *services.OrderService

	jwtSecret []byte

	idempotencyCache map[string]*IdempotencyRecord
}

type IdempotencyRecord struct {
	Response   []byte
	StatusCode int
	Expiry     time.Time
}

func NewHandler(
	authService *services.AuthService,
	itemService *services.ItemService,
	cartService *services.CartService,
	orderService *services.OrderService,
	jwtSecret []byte,
) *Handler {
	return &Handler{
		authService:      authService,
		itemService:      itemService,
		cartService:      cartService,
		orderService:     orderService,
		jwtSecret:        jwtSecret,
		idempotencyCache: make(map[string]*IdempotencyRecord),
	}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// --------------------
	// CART
	// --------------------

	mux.Handle(
		"GET /user/cart",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.GetUserCart,
			),
			h.jwtSecret,
		),
	)

	mux.Handle(
		"GET /user/cart/cursor",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.GetUserCartCursor,
			),
			h.jwtSecret,
		),
	)
	mux.Handle(
		"PATCH /user/cart/items/{item_id}",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.UpsertCartItem,
			),
			h.jwtSecret,
		),
	)

	mux.Handle(
		"DELETE /user/cart/items/{item_id}",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.RemoveCartItem,
			),
			h.jwtSecret,
		),
	)

	// --------------------
	// ORDERS
	// --------------------

	mux.Handle(
		"POST /orders",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.CreateOrder,
			),
			h.jwtSecret,
		),
	)

	mux.Handle(
		"GET /user/orders",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.GetUserOrders,
			),
			h.jwtSecret,
		),
	)
	mux.Handle(
		"GET /user/orders/cursor",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.GetUserOrdersCursor,
			),
			h.jwtSecret,
		),
	)
	// --------------------
	// ITEMS
	// --------------------

	mux.HandleFunc(
		"GET /items",
		h.GetItems,
	)

	mux.HandleFunc(
		"GET /items/cursor",
		h.GetItemsCursor,
	)

	mux.HandleFunc(
		"GET /items/{item_id}",
		h.GetItemByID,
	)

	// --------------------
	// AUTH
	// --------------------

	mux.HandleFunc(
		"POST /signup",
		h.CreateUser,
	)

	mux.HandleFunc(
		"POST /login",
		h.LoginUser,
	)

	mux.HandleFunc(
		"GET /token",
		h.IssueJWT,
	)

	return mux
}
func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}
func intQuery(
	r *http.Request,
	name string,
	def int,
) int {
	value :=
		r.URL.Query().
			Get(name)

	if value == "" {
		return def
	}

	n, err :=
		strconv.Atoi(value)

	if err != nil {
		return def
	}

	return n
}
