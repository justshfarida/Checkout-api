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

// CreateOrder places an order from the user's cart.
// @Summary Create an order
// @Description Creates an order from the authenticated user's cart. Safe to retry: pass the same Idempotency-Key with the same body to avoid duplicate orders.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Idempotency-Key header string true "Client-generated idempotency key"
// @Success 201 {object} service.PlaceOrderResult
// @Success 402 {object} service.PlaceOrderResult "Payment required — payment failed"
// @Failure 400 {object} httperror.Envelope
// @Failure 401 {object} httperror.Envelope
// @Router /user/orders [post]
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

// GetOrdersByUserID lists the current user's orders.
// @Summary List the current user's orders
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Order
// @Failure 401 {object} httperror.Envelope
// @Router /user/orders [get]
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
