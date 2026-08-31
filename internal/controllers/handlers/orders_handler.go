package handlers

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/models"
	"checkout-api/internal/service"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

type OrdersService interface {
	PlaceOrder(ctx context.Context, userID int, idempotencyKey, requestHash string) (*service.PlaceOrderResult, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error)
}

type OrdersHandler struct {
	service OrdersService
}

func NewOrdersHandler(service OrdersService) *OrdersHandler {
	return &OrdersHandler{service: service}
}

func (handler *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		WriteUnauthorized(w, "unauthorized")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		WriteValidationError(w, "Idempotency-Key header is required")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteValidationError(w, "invalid request body")
		return
	}
	bodyHash := sha256.Sum256(bodyBytes)
	requestHash := hex.EncodeToString(bodyHash[:])

	result, err := handler.service.PlaceOrder(ctx, userID, idempotencyKey, requestHash)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	writeJSON(w, result.StatusCode, result.Body)
}

func (handler *OrdersHandler) GetOrdersByUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w, "unauthorized")
		return
	}

	orders, err := handler.service.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}
