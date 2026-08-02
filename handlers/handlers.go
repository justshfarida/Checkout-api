package handlers

import (
	"checkout-api/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ItemStore defines the data operations the handler needs.
type ItemStore interface {
	GetItems() []*models.Item
	GetItem(id int) *models.Item
	CreateOrder(userID int, items []models.LineItem, total int, status string) *models.Order
	CreateUserCart(cart *models.Cart)
	GetUserCart(userID int) *models.Cart
	UpdateQuantityOfItem(userID int, itemID int, quantity int) (*models.Cart, error)
	RemoveItemFromCart(userID int, itemID int) (*models.Cart, bool, error)
	DeleteUserCart(userID int)
}

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store            ItemStore
	idempotencyStore map[string]IdempotencyRecord
	idempotencyMutex sync.Mutex
}
type IdempotencyRecord struct {
	StatusCode   int
	ResponseBody []byte
	CreatedAt    time.Time
	RequestHash  string //Store the SHA-256 hash of the request body
}

// NewHandler creates a Handler with the given store.
func NewHandler(s ItemStore) *Handler {
	return &Handler{store: s,
		idempotencyStore: make(map[string]IdempotencyRecord),
	}
}

// CreateOrderRequest is the payload for POST /orders.
type CreateOrderRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

// PaymentResult represents a response from the payment provider.
type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

// mockProcessPayment simulates a payment provider call.
func mockProcessPayment(amount int) PaymentResult {
	if amount > 0 && amount < 1000000 {
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

// GetItems handles GET /items — returns all available items.
func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items := h.store.GetItems()
	writeJSON(w, http.StatusOK, items)
}

// GetItemByID handles GET /items/{id} — returns a single item.
func (h *Handler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item := h.store.GetItem(id)
	if item == nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, item)
}
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	// Read request body and calculate SHA-256 hash
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256(bodyBytes)
	currentHash := hex.EncodeToString(hash[:])

	// Check cached idempotency response
	h.idempotencyMutex.Lock()
	if record, exists := h.idempotencyStore[idempotencyKey]; exists {
		if time.Since(record.CreatedAt) < 24*time.Hour {
			h.idempotencyMutex.Unlock()

			if record.RequestHash != currentHash {
				http.Error(w, "idempotency key reused with a different request payload", http.StatusConflict)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			w.Write(record.ResponseBody)
			return
		}
	}
	h.idempotencyMutex.Unlock()

	// Decode request from the bytes we already read
	var req CreateOrderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cart := h.store.GetUserCart(req.UserID)
	if cart == nil || len(cart.Items) == 0 {
		http.Error(w, "cart is empty or does not exist", http.StatusBadRequest)
		return
	}

	// Calculate total
	total := 0
	for _, item := range cart.Items {
		total += item.Price * item.Quantity
	}

	// Process payment
	paymentResult := mockProcessPayment(total)

	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	order := h.store.CreateOrder(req.UserID, cart.Items, total, status)

	var (
		statusCode  int
		respPayload any
	)

	if paymentResult.Success {
		statusCode = http.StatusCreated
		respPayload = map[string]any{
			"order":   order,
			"payment": paymentResult,
		}

		// Delete cart after successful checkout
		h.store.DeleteUserCart(req.UserID)
	} else {
		statusCode = http.StatusPaymentRequired
		respPayload = map[string]any{
			"order":   order,
			"payment": paymentResult,
		}
	}

	respBytes, _ := json.Marshal(respPayload)

	// Store response for idempotency
	h.idempotencyMutex.Lock()
	h.idempotencyStore[idempotencyKey] = IdempotencyRecord{
		StatusCode:   statusCode,
		ResponseBody: respBytes,
		RequestHash:  currentHash,
		CreatedAt:    time.Now(),
	}
	h.idempotencyMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(respBytes)
}

// CreateOrder handles POST /orders — creates an order with mock payment.
// func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	var req CreateOrderRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	// Validate and calculate total
// 	total := 0
// 	orderItems := make([]models.LineItem, 0, len(req.Items))
// 	for _, item := range req.Items {
// 		storeItem := h.store.GetItem(item.ItemID)
// 		if storeItem == nil {
// 			http.Error(w, fmt.Sprintf("Item %d not found", item.ItemID), http.StatusBadRequest)
// 			return
// 		}
// 		itemTotal := storeItem.Price * item.Quantity
// 		total += itemTotal
// 		orderItems = append(orderItems, models.LineItem{
// 			ItemID:   item.ItemID,
// 			Quantity: item.Quantity,
// 			Price:    storeItem.Price,
// 		})
// 	}

// 	// Process payment (mock)
// 	paymentResult := mockProcessPayment(total)

// 	status := "paid"
// 	if !paymentResult.Success {
// 		status = "failed"
// 	}

// 	order := h.store.CreateOrder(req.UserID, orderItems, total, status)

// 	if paymentResult.Success {
// 		writeJSON(w, http.StatusCreated, map[string]any{
// 			"order":   order,
// 			"payment": paymentResult,
// 		})
// 	} else {
// 		writeJSON(w, http.StatusPaymentRequired, map[string]any{
// 			"order":   order,
// 			"payment": paymentResult,
// 		})
// 	}
// }

// writeJSON encodes v as JSON and writes it to the response.
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

func (h *Handler) CreateUserCartAndAddItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.store.GetUserCart(req.UserID) != nil {
		http.Error(w, "cart already exists", http.StatusConflict)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "can't create empty cart", http.StatusBadRequest)
		return
	}

	orderItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem := h.store.GetItem(item.ItemID)
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
		ID:     fmt.Sprintf("cart_%d", time.Now().UnixNano()),
		UserID: req.UserID,
		Items:  orderItems,
	}

	h.store.CreateUserCart(userCart)
	writeJSON(w, http.StatusCreated, userCart)
}

func (h *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {
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

	cart := h.store.GetUserCart(userID)
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

type UpdateItemQuantityRequest struct {
	UserID   int `json:"user_id"`
	Quantity int `json:"quantity"`
}

func (h *Handler) UpdateQuantityOfItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/user/cart/items/")
	fmt.Println(r.URL.Path)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	var req UpdateItemQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	updatedCart, err := h.store.UpdateQuantityOfItem(req.UserID, id, req.Quantity)
	if err != nil {
		if err.Error() == "cart not found" || err.Error() == "item not found in cart" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, updatedCart)
}

type RemoveItemFromCartRequest struct {
	UserID int `json:"user_id"`
}

// DELETE /user/cart/items/{id}
func (h *Handler) RemoveItemFromCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/user/cart/items/")
	itemID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	var req RemoveItemFromCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	_, _, err = h.store.RemoveItemFromCart(req.UserID, itemID)
	if err != nil {
		if err.Error() == "cart not found" || err.Error() == "item not found in cart" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 204 No Content output
	w.WriteHeader(http.StatusNoContent)
}
