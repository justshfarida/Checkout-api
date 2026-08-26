package handlers

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Store interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)

	CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
	UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error)
	RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error)

	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error)

	SignUp(ctx context.Context, username string, password string) (*models.User, error)
	Login(ctx context.Context, username, password string) (*models.User, error)
	SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error

	SaveIdempotencyRecord(ctx context.Context, key, requestHash string, statusCode int, body any) error
	GetIdempotencyRecord(ctx context.Context, key string) (*models.IdempotencyRecord, error)
}

type Handler struct {
	store       Store
	authManager *auth.Manager
}

func NewHandler(s Store, authManager *auth.Manager) *Handler {
	return &Handler{store: s, authManager: authManager}
}

// Items

func (handler *Handler) GetItems(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.store.GetItems(request.Context())
	if err != nil {
		http.Error(writer, "failed to fetch items", http.StatusInternalServerError)
		return
	}
	query := request.URL.Query()
	category := query.Get("category")
	sortBy := query.Get("sort")

	if category != "" {
		filtered := make([]*models.Item, 0, len(items))
		for _, item := range items {
			if item.Category == category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	switch sortBy {
	case "price":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Price < items[j].Price
		})
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
	}

	writer.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(writer, http.StatusOK, items)
}

func (handler *Handler) GetItem(writer http.ResponseWriter, request *http.Request) {
	fullUrl := request.URL.Path
	UrlID := strings.TrimPrefix(fullUrl, "/items/")
	idNum, err := strconv.Atoi(UrlID)

	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	item, err := handler.store.GetItem(request.Context(), idNum)
	if err != nil {
		http.Error(writer, http.StatusText(404), http.StatusNotFound)
		return
	}

	writeJSON(writer, http.StatusOK, item)
}

// Cart

func (handler *Handler) CreateUserCartAndAddItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateUserCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "can't create empty cart", http.StatusBadRequest)
		return
	}

	orderItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem, err := handler.store.GetItem(ctx, item.ItemID)
		if err != nil {
			http.Error(w, fmt.Sprintf("item %d not found", item.ItemID), http.StatusBadRequest)
			return
		}
		orderItems = append(orderItems, models.LineItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	userCart := &models.Cart{
		ID:     fmt.Sprintf("cart_%d", userID),
		UserID: userID, // use the authenticated user, not the request body — closes the earlier-flagged trust gap too
		Items:  orderItems,
	}

	if err := handler.store.CreateUserCartAndAddItems(ctx, userCart); err != nil {
		http.Error(w, "failed to create cart", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, userCart)
}

func (handler *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	cart, err := handler.store.GetUserCart(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to fetch cart", http.StatusInternalServerError)
		return
	}
	if cart == nil {
		cart = &models.Cart{ID: "", UserID: userID, Items: []models.LineItem{}}
	}
	writeJSON(w, http.StatusOK, cart)
}

func (handler *Handler) UpdateItemQuantity(writer http.ResponseWriter, request *http.Request) {
	idStr := request.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req updateQuantityRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		http.Error(writer, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(writer, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	ctx := request.Context()
	found, err := handler.store.UpdateCartItem(ctx, userID, id, req.Quantity)
	if err != nil {
		http.Error(writer, "failed to update cart item", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	cart, err := handler.store.GetUserCart(ctx, userID)
	if err != nil {
		http.Error(writer, "failed to fetch updated cart", http.StatusInternalServerError)
		return
	}
	writeJSON(writer, http.StatusOK, cart)
}

func (handler *Handler) DeleteItem(writer http.ResponseWriter, request *http.Request) {
	idStr := request.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := request.Context()
	found, err := handler.store.RemoveCartItem(ctx, userID, id)
	if err != nil {
		http.Error(writer, "failed to remove cart item", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	cart, err := handler.store.GetUserCart(ctx, userID)
	if err != nil {
		http.Error(writer, "failed to fetch cart", http.StatusInternalServerError)
		return
	}
	if cart == nil || len(cart.Items) == 0 {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(writer, http.StatusOK, cart)
}

// Orders

func (handler *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	bodyHash := sha256.Sum256(bodyBytes)
	currentHash := hex.EncodeToString(bodyHash[:])

	cached, err := handler.store.GetIdempotencyRecord(ctx, idempotencyKey)
	if err != nil {
		http.Error(w, "failed to check idempotency", http.StatusInternalServerError)
		return
	}
	if cached != nil {
		if cached.RequestHash != currentHash {
			http.Error(w, "Idempotency-Key already used with a different request body", http.StatusConflict)
			return
		}
		writeJSON(w, cached.StatusCode, cached.Body)
		return
	}

	cart, err := handler.store.GetUserCart(ctx, userID)
	if err != nil {
		http.Error(w, "failed to fetch cart", http.StatusInternalServerError)
		return
	}
	if cart == nil || len(cart.Items) == 0 {
		http.Error(w, "cart is empty or does not exist", http.StatusBadRequest)
		return
	}

	total := 0
	orderItems := make([]models.LineItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		storeItem, err := handler.store.GetItem(ctx, item.ItemID)
		if err != nil {
			http.Error(w, fmt.Sprintf("item %d not found", item.ItemID), http.StatusBadRequest)
			return
		}
		if item.Quantity <= 0 {
			http.Error(w, fmt.Sprintf("invalid quantity for item %d", item.ItemID), http.StatusBadRequest)
			return
		}
		if storeItem.Stock < item.Quantity {
			http.Error(w, fmt.Sprintf("insufficient stock for item %d", item.ItemID), http.StatusConflict)
			return
		}
		total += storeItem.Price * item.Quantity
		orderItems = append(orderItems, models.LineItem{
			ItemID: item.ItemID, Quantity: item.Quantity, Price: storeItem.Price,
		})
	}

	var paymentResult PaymentResult
	success := retryWithBackoff(3, func() (bool, error) {
		paymentResult = mockProcessPayment(total)
		return paymentResult.Success, nil
	})

	if !success {
		if err := handler.store.SaveIdempotencyRecord(ctx, idempotencyKey, currentHash, http.StatusPaymentRequired, paymentResult); err != nil {
			http.Error(w, "failed to save idempotency record", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"payment": paymentResult})
		return
	}

	order, err := handler.store.CreateOrder(ctx, userID, orderItems, total, "paid")
	if err != nil {
		http.Error(w, "failed to create order", http.StatusInternalServerError)
		return
	}
	if err := handler.store.DeleteUserCart(ctx, userID); err != nil {
		http.Error(w, "failed to clear cart", http.StatusInternalServerError)
		return
	}

	responseBody := map[string]any{"order": order, "payment": paymentResult}
	if err := handler.store.SaveIdempotencyRecord(ctx, idempotencyKey, currentHash, http.StatusCreated, responseBody); err != nil {
		http.Error(w, "failed to save idempotency record", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, responseBody)
}

func (handler *Handler) GetOrdersByUserID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/users/"), "/orders")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	orders, err := handler.store.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to fetch orders", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// Users

func (handler *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to process password", http.StatusInternalServerError)
		return
	}

	user, err := handler.store.SignUp(ctx, req.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (handler *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	user, err := handler.store.Login(ctx, req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	accessToken, err := handler.authManager.GenerateAccessToken(user.ID)

	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	refreshToken, err := handler.authManager.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	if err := handler.store.SaveRefreshToken(ctx, refreshToken, user.ID, time.Now().Add(30*24*time.Hour)); err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func (handler *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	oldToken := r.Header.Get("Refresh-Token")
	if oldToken == "" {
		http.Error(w, "missing Refresh-Token header", http.StatusBadRequest)
		return
	}

	rt, err := handler.store.GetRefreshToken(ctx, oldToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	if rt.Revoked {
		http.Error(w, "refresh token has been revoked", http.StatusUnauthorized)
		return
	}
	if time.Now().After(rt.ExpiresAt) {
		http.Error(w, "refresh token expired", http.StatusUnauthorized)
		return
	}

	if err := handler.store.RevokeRefreshToken(ctx, oldToken); err != nil {
		http.Error(w, "failed to rotate token", http.StatusInternalServerError)
		return
	}

	newAccessToken, err := handler.authManager.GenerateAccessToken(rt.UserID)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}
	newRefreshToken, err := handler.authManager.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}
	if err := handler.store.SaveRefreshToken(ctx, newRefreshToken, rt.UserID, time.Now().Add(30*24*time.Hour)); err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	})
}

// Retry  with backoff
func retryWithBackoff(maxRetries int, fn func() (bool, error)) bool {
	baseDelay := 100 * time.Millisecond
	maxDelay := 400 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		ok, err := fn()
		if err == nil && ok {
			return true
		}

		if attempt < maxRetries {
			delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
			if delay > maxDelay {
				delay = maxDelay
			}
			time.Sleep(delay)
		}
	}
	return false
}

func mockProcessPayment(amount int) PaymentResult {
	if amount > 0 && amount < 1000000 {
		return PaymentResult{
			Success: true,
		}
	}
	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
