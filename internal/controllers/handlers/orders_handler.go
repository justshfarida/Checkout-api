package handlers

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error)
}

type OrdersHandler struct {
	service OrderService
}

func NewOrdersHandler(service OrderService) *OrdersHandler {
	return &OrdersHandler{service: service}
}

func (handler *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
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

	cached, err := handler.service.GetIdempotencyRecord(ctx, idempotencyKey)
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

	cart, err := handler.service.GetUserCart(ctx, userID)
	if err != nil {
		http.Error(w, "failed to fetch cart", http.StatusInternalServerError)
		return
	}
	if cart == nil || len(cart.Items) == 0 {
		http.Error(w, "cart is empty or does not exist", http.StatusBadRequest)
		return
	}

	// BUSINESS LOGIC
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
	//
	var paymentResult dtos.PaymentResult
	success := retryWithBackoff(3, func() (bool, error) {
		paymentResult = mockProcessPayment(total)
		return paymentResult.Success, nil
	})

	if !success {
		if err := handler.service.SaveIdempotencyRecord(ctx, idempotencyKey, currentHash, http.StatusPaymentRequired, paymentResult); err != nil {
			http.Error(w, "failed to save idempotency record", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"payment": paymentResult})
		return
	}

	order, err := handler.service.CreateOrder(ctx, userID, orderItems, total, "paid")
	if err != nil {
		http.Error(w, "failed to create order", http.StatusInternalServerError)
		return
	}
	if err := handler.service.DeleteUserCart(ctx, userID); err != nil {
		http.Error(w, "failed to clear cart", http.StatusInternalServerError)
		return
	}

	responseBody := map[string]any{"order": order, "payment": paymentResult}
	if err := handler.service.SaveIdempotencyRecord(ctx, idempotencyKey, currentHash, http.StatusCreated, responseBody); err != nil {
		http.Error(w, "failed to save idempotency record", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, responseBody)
}

func (handler *OrdersHandler) GetOrdersByUserID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/users/"), "/orders")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	orders, err := handler.service.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to fetch orders", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}
