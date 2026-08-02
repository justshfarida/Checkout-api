package handlers

import (
	"checkout-api/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ItemStore interface {
	GetItems() []*models.Item
	GetItem(id int) *models.Item
	CreateOrder(userID int, items []models.LineItem, total int, status string) *models.Order
	CreateUserCart(cart *models.Cart)
	GetUserCart(userID int) *models.Cart
	GetOrdersByUserID(userID int) []*models.Order
	UpdateItemQuantity(userID int, itemID int, quantity int) *models.Cart
	DeleteItem(userID int, itemID int) (*models.Cart, bool, bool)
	SaveIdempotencyRecord(key, requestHash string, statusCode int, body any)
	GetIdempotencyRecord(key string) *models.IdempotencyRecord
	DeleteCart(userID int)
	//PlaceOrder(userID int, items []models.LineItem, total int, status string) *models.Order
}

type createOrderItem struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}
type createOrderRequest struct {
	UserID int               `json:"user_id"`
	Items  []createOrderItem `json:"items"`
}
type Handler struct{ store ItemStore }

func NewHandler(s ItemStore) *Handler { return &Handler{store: s} }

type CreateOrderRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}
type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

func mockProcessPayment(amount int) PaymentResult {
	if amount > 0 && amount < 1000000 {
		return PaymentResult{
			Success: true,
			// TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
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

type CreateUserCartRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

func (handler *Handler) CreateUserCartAndAddItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if handler.store.GetUserCart(req.UserID) != nil {
		http.Error(w, "cart already exists", http.StatusConflict)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "can't create empty cart", http.StatusBadRequest)
		return
	}

	orderItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem := handler.store.GetItem(item.ItemID)
		if storeItem == nil {
			http.Error(w, fmt.Sprintf("Item %d not found", item.ItemID), http.StatusBadRequest)
			return
		}
		orderItems = append(orderItems, models.LineItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	userCart := &models.Cart{
		ID:     fmt.Sprintf("cart_%d", req.UserID),
		UserID: req.UserID,
		Items:  orderItems,
	}

	handler.store.CreateUserCart(userCart)
	writeJSON(w, http.StatusCreated, userCart)
}

func (handler *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	cart := handler.store.GetUserCart(userID)
	if cart == nil {
		emptyCart := &models.Cart{
			ID:     "",
			UserID: userID,
			Items:  []models.LineItem{},
		}
		writeJSON(w, http.StatusOK, emptyCart)
		return
	}

	writeJSON(w, http.StatusOK, cart)
}

// -------------------------------------------------------------------------

// LECTURE 1 - TASK 1
func (handler *Handler) GetItem(writer http.ResponseWriter, request *http.Request) {
	fullUrl := request.URL.Path
	UrlID := strings.TrimPrefix(fullUrl, "/items/")
	idNum, err := strconv.Atoi(UrlID)

	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	} // bad request aka value isnt num test passed

	item := handler.store.GetItem(idNum)
	if item == nil {
		http.Error(writer, http.StatusText(404), http.StatusNotFound)
		return
	} // item not found test passed

	writeJSON(writer, http.StatusOK, item)
}

// LECTURE 1 - TASK 2
func (handler *Handler) GetOrdersByUserID(writer http.ResponseWriter, request *http.Request) {
	fullUrl := request.URL.Path
	UrlID := strings.TrimPrefix(fullUrl, "/users/")
	IDNum := strings.TrimSuffix(UrlID, "/orders")
	num, err := strconv.Atoi(IDNum)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	orders := handler.store.GetOrdersByUserID(num)
	writeJSON(writer, http.StatusOK, orders)
}

// LECTURE 1 - TASK 3
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

// LECTURE 1 - TASK 3 (retry with backoff)

// LECTURE 1 - TASK 1
func (handler *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {

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

	if cached := handler.store.GetIdempotencyRecord(idempotencyKey); cached != nil {
		if cached.RequestHash != currentHash {
			http.Error(w, "Idempotency-Key already used with a different request body", http.StatusConflict)
			return
		}
		writeJSON(w, cached.StatusCode, cached.Body)
		return
	}

	var req CreateOrderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cart := handler.store.GetUserCart(req.UserID)
	if cart == nil || len(cart.Items) == 0 {
		http.Error(w, "cart is empty or does not exist", http.StatusBadRequest)
		return
	}

	total := 0
	orderItems := make([]models.LineItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		storeItem := handler.store.GetItem(item.ItemID)
		if storeItem == nil {
			http.Error(w, fmt.Sprintf("Item %d not found", item.ItemID), http.StatusBadRequest)
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
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	var paymentResult PaymentResult
	success := retryWithBackoff(3, func() (bool, error) {
		paymentResult = mockProcessPayment(total)
		return paymentResult.Success, nil
	})

	if !success {
		handler.store.SaveIdempotencyRecord(idempotencyKey, currentHash, http.StatusPaymentRequired, paymentResult)
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"payment": paymentResult})
		return
	}

	order := handler.store.CreateOrder(req.UserID, orderItems, total, "paid")
	handler.store.DeleteCart(req.UserID)

	responseBody := map[string]any{"order": order, "payment": paymentResult}
	handler.store.SaveIdempotencyRecord(idempotencyKey, currentHash, http.StatusCreated, responseBody)
	writeJSON(w, http.StatusCreated, responseBody)
}

func mockChargePayment(amountCents int) bool {
	return rand.Intn(10) < 6
} // for failing before backoff

// LECTURE 1 - TASK 4
func (handler *Handler) GetItems(writer http.ResponseWriter, request *http.Request) {
	items := handler.store.GetItems()

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

/* func mockChargePayment(amountCents int) bool {
	return rand.Intn(10) < 6
}
*/
// -------------------------------------------------------------------------

// LECTURE 2 - TASK 1
type updateQuantityRequest struct {
	UserID   int `json:"user_id"`
	Quantity int `json:"quantity"`
}

func (handler *Handler) UpdateItemQuantity(writer http.ResponseWriter, request *http.Request) {
	idStr := request.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
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

	cart := handler.store.UpdateItemQuantity(req.UserID, id, req.Quantity)
	if cart == nil {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	writeJSON(writer, http.StatusOK, cart)
}

// LECTURE 2 - TASK 2
type deleteItemRequest struct {
	UserID int `json:"user_id"`
}

func (handler *Handler) DeleteItem(writer http.ResponseWriter, request *http.Request) {
	idStr := request.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	var req deleteItemRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		http.Error(writer, "invalid request body", http.StatusBadRequest)
		return
	}

	cart, cartDeleted, itemFound := handler.store.DeleteItem(req.UserID, id)
	if !itemFound {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if cartDeleted {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(writer, http.StatusOK, cart)
}
