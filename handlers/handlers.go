package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"checkout-api/middlewares"
	"checkout-api/models"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ItemStore defines the data operations the handler needs.
type ItemStore interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID int, status string) error
	UpsertCartItem(ctx context.Context, userID int, itemID int, quantity int) error
	GetUserCart(ctx context.Context, userID int) ([]models.CartItem, error)
	DeleteUserCart(ctx context.Context, userID int) error
	RemoveCartItem(ctx context.Context, userID int, itemID int) error
	SaveUser(ctx context.Context, email string, hash []byte) error
	FindUserByEmail(ctx context.Context, email string) (models.User, error)
	SaveRefreshToken(
		ctx context.Context,
		userID int,
		tokenHash []byte,
		expiresAt time.Time,
	) error
	RotateRefreshToken(
		ctx context.Context,
		oldTokenHash []byte,
		newTokenHash []byte,
		newExpiresAt time.Time,
	) (int, error)
	GetOrdersByUserID(
		ctx context.Context,
		userID int,
	) ([]models.UserOrder, error)
}

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store            ItemStore
	idempotencyCache map[string]*IdempotencyRecord
	jwtSecret        []byte
}

// NewHandler creates a Handler with the given store.
func NewHandler(s ItemStore, jwtSecret []byte) *Handler {
	return &Handler{
		store:            s,
		idempotencyCache: make(map[string]*IdempotencyRecord),
		jwtSecret:        jwtSecret,
	}
}

type IdempotencyRecord struct {
	Response   []byte
	StatusCode int
	Expiry     time.Time
}

// mockProcessPayment simulates a payment provider call.
func mockProcessPayment(amount int) PaymentResult {
	if amount > 0 && amount < 40 {
		return PaymentResult{
			Success:       true,
			TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		}
	}
	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}

func (h *Handler) UpsertCartItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// user is now authenticated
	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "missing item_id", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be integer",
		})
		return
	}

	var req UpsertCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	if err := h.store.UpsertCartItem(r.Context(), userID, itemID, req.Quantity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	// userIDStr := r.Header.Get("X-User-ID")
	// if userIDStr == "" {
	// 	http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
	// 	return
	// }

	// userID, err := strconv.Atoi(userIDStr)
	// if err != nil {
	// 	http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
	// 	return
	// }
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "missing item_id", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be integer",
		})
		return
	}

	if err := h.store.RemoveCartItem(r.Context(), userID, itemID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {

	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cart, err := h.store.GetUserCart(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	items := make([]CartItemResponse, 0, len(cart))
	for _, item := range cart {
		items = append(items, CartItemResponse{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Stock:       item.Stock,
			CreatedAt:   item.CreatedAt,
			Quantity:    item.Quantity,
		})
	}

	writeJSON(w, http.StatusOK, CartResponse{
		UserID: userID,
		Items:  items,
	})
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	if record, exists := h.idempotencyCache[idempotencyKey]; exists {
		if time.Now().Before(record.Expiry) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			w.Write(record.Response)
			return
		}
		delete(h.idempotencyCache, idempotencyKey)
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.LineItems) == 0 {
		http.Error(w, "items must not be empty", http.StatusBadRequest)
		return
	}

	items := make([]models.LineItem, 0, len(req.LineItems))
	for _, i := range req.LineItems {
		items = append(items, models.LineItem{
			ItemID:   i.ItemID,
			Quantity: i.Quantity,
			Price:    i.Price,
		})
	}

	order, err := h.store.CreateOrder(r.Context(), userID, items, req.Total, "pending")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	paymentResult := mockProcessPayment(req.Total)

	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	if err := h.store.UpdateOrderStatus(r.Context(), order.ID, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	order.Status = status

	responseData := map[string]any{
		"order":   order,
		"payment": paymentResult,
	}

	statusCode := http.StatusCreated
	if !paymentResult.Success {
		statusCode = http.StatusPaymentRequired
	}

	responseBody, _ := json.Marshal(responseData)
	h.idempotencyCache[idempotencyKey] = &IdempotencyRecord{
		Response:   responseBody,
		StatusCode: statusCode,
		Expiry:     time.Now().Add(24 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(responseBody)
}

// GetItems handles GET /items — returns all available items.
func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.GetItems(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, items)
	return
}

// GetItemByID handles GET /items/{id} — returns a single i`tem.
func (h *Handler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be an integer",
		})
		return
	}

	item, err := h.store.GetItem(r.Context(), itemID)
	if err != nil {
		fmt.Printf("error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if item == nil {
		writeJSON(w, http.StatusNotFound, nil)
		return
	}

	writeJSON(w, http.StatusOK, item)
	return
}

// writeJSON encodes v as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) LoginUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AuthRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid body",
			http.StatusBadRequest,
		)
		return
	}

	user, err :=
		h.store.FindUserByEmail(
			r.Context(),
			req.Email,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			writeJSON(
				w,
				http.StatusUnprocessableEntity,
				ErrorMessageResponse{
					Message: "user does not exist",
				},
			)
			return
		}

		fmt.Printf(
			"cannot query %q\n",
			err.Error(),
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		user.Hash,
		[]byte(req.Password),
	)

	if err != nil {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	// --------------------
	// ACCESS TOKEN
	// --------------------

	accessExpiresAt :=
		time.Now().Add(
			15 * time.Minute,
		)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				accessExpiresAt,
			),

			Subject: strconv.Itoa(
				user.ID,
			),

			IssuedAt: jwt.NewNumericDate(
				time.Now(),
			),
		},
	)

	signedJWT, err :=
		token.SignedString(
			h.jwtSecret,
		)

	if err != nil {
		fmt.Printf(
			"cannot generate signed JWT: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	// --------------------
	// REFRESH TOKEN
	// --------------------

	refreshToken, err :=
		generateRefreshToken()

	if err != nil {
		fmt.Printf(
			"cannot generate refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	refreshTokenHash :=
		hashRefreshToken(
			refreshToken,
		)

	refreshTokenExpiresAt :=
		time.Now().Add(
			30 * 24 * time.Hour,
		)

	err =
		h.store.SaveRefreshToken(
			r.Context(),
			user.ID,
			refreshTokenHash,
			refreshTokenExpiresAt,
		)

	if err != nil {
		fmt.Printf(
			"cannot save refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	// --------------------
	// HTTPONLY COOKIE
	// --------------------

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,

			// localhost uses HTTP.
			// In production with HTTPS,
			// change this to true.
			Secure: false,

			SameSite: http.SameSiteLaxMode,

			Expires: refreshTokenExpiresAt,

			MaxAge: 30 * 24 * 60 * 60,
		},
	)

	// Frontend receives ONLY access JWT.
	writeJSON(
		w,
		http.StatusOK,
		AuthResponse{
			JWT: signedJWT,
		},
	)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// validate email
	_, err = mail.ParseAddress(req.Email)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "invalid email",
		})
		return
	}

	pwlen := len(req.Password)
	// validate password
	if pwlen < 12 || pwlen > 25 {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "password is too short or too long",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.store.SaveUser(r.Context(), req.Email, hash)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, nil)
}

func (h *Handler) IssueJWT(
	w http.ResponseWriter,
	r *http.Request,
) {
	// --------------------
	// READ REFRESH TOKEN COOKIE
	// --------------------

	cookie, err :=
		r.Cookie(
			"refresh_token",
		)

	if err != nil {
		if errors.Is(
			err,
			http.ErrNoCookie,
		) {
			http.Error(
				w,
				"Refresh token is missing",
				http.StatusUnauthorized,
			)
			return
		}

		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	oldRefreshToken :=
		cookie.Value

	if oldRefreshToken == "" {
		http.Error(
			w,
			"Refresh token is missing",
			http.StatusUnauthorized,
		)
		return
	}

	// --------------------
	// GENERATE NEW REFRESH TOKEN
	// --------------------

	newRefreshToken, err :=
		generateRefreshToken()

	if err != nil {
		fmt.Printf(
			"failed to generate refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	oldTokenHash :=
		hashRefreshToken(
			oldRefreshToken,
		)

	newTokenHash :=
		hashRefreshToken(
			newRefreshToken,
		)

	newRefreshExpiresAt :=
		time.Now().Add(
			30 * 24 * time.Hour,
		)

	// --------------------
	// ROTATE REFRESH TOKEN
	// --------------------

	userID, err :=
		h.store.RotateRefreshToken(
			r.Context(),
			oldTokenHash,
			newTokenHash,
			newRefreshExpiresAt,
		)

	if err != nil {
		fmt.Printf(
			"failed to rotate refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Invalid or expired refresh token",
			http.StatusUnauthorized,
		)
		return
	}

	// --------------------
	// GENERATE NEW ACCESS JWT
	// --------------------

	accessExpiresAt :=
		time.Now().Add(
			15 * time.Minute,
		)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Subject: strconv.Itoa(
				userID,
			),

			IssuedAt: jwt.NewNumericDate(
				time.Now(),
			),

			ExpiresAt: jwt.NewNumericDate(
				accessExpiresAt,
			),
		},
	)

	signedJWT, err :=
		token.SignedString(
			h.jwtSecret,
		)

	if err != nil {
		fmt.Printf(
			"failed to sign JWT: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	// --------------------
	// REPLACE OLD COOKIE
	// --------------------

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    newRefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,

			SameSite: http.SameSiteLaxMode,

			Expires: newRefreshExpiresAt,

			MaxAge: 30 * 24 * 60 * 60,
		},
	)

	// Return ONLY new access JWT.
	writeJSON(
		w,
		http.StatusOK,
		AuthResponse{
			JWT: signedJWT,
		},
	)
}
func generateRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf(
			"failed to generate refresh token: %w",
			err,
		)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}
func hashRefreshToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}
func (h *Handler) GetUserOrders(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middlewares.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)

		return
	}

	orders, err :=
		h.store.GetOrdersByUserID(
			r.Context(),
			userID,
		)

	if err != nil {
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)

		return
	}

	response := make(
		[]UserOrderResponse,
		0,
		len(orders),
	)

	for _, order := range orders {
		items := make(
			[]OrderItemResponse,
			0,
			len(order.Items),
		)

		for _, item := range order.Items {
			items = append(
				items,
				OrderItemResponse{
					ItemID:   item.ItemID,
					Name:     item.Name,
					Quantity: item.Quantity,
					Price:    item.Price,
				},
			)
		}

		response = append(
			response,
			UserOrderResponse{
				ID:     order.ID,
				Items:  items,
				Total:  order.Total,
				Status: order.Status,
			},
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}
