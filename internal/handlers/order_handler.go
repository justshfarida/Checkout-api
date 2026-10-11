package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/models"
	"checkout-api/internal/services"
)

func (h *Handler) CreateOrder(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middlewares.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication is required",
		)
		return
	}

	idempotencyKey :=
		r.Header.Get(
			"Idempotency-Key",
		)

	if idempotencyKey == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"IDEMPOTENCY_KEY_REQUIRED",
			"Idempotency-Key header is required",
		)
		return
	}

	// Keys are scoped per user: the same key sent by another user is a
	// different request and must never replay someone else's order.
	cacheKey := strconv.Itoa(userID) + ":" + idempotencyKey

	// The lock only guards the cache map and is held for microseconds at a
	// time. The slow part (creating the order and calling Xsolla) runs with
	// no lock held, so requests with different keys run in parallel. An
	// "in progress" marker still stops duplicate requests with the same key.

	// STEP 1: look up the key and claim it.
	h.idempotencyMu.Lock()

	if record, exists := h.idempotencyCache[cacheKey]; exists {
		switch {
		case record.InProgress:
			// The same request is still being processed (double-click or a
			// fast client retry). Don't start a second order.
			h.idempotencyMu.Unlock()
			writeError(
				w,
				http.StatusConflict,
				"REQUEST_IN_PROGRESS",
				"This order is already being processed, please wait",
			)
			return

		case time.Now().Before(record.Expiry):
			// Finished earlier: replay the saved response, no new order.
			h.idempotencyMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			_, _ = w.Write(record.Response)
			return
		}
		// Expired: treat it as a new request below.
	}

	h.idempotencyCache[cacheKey] = &IdempotencyRecord{InProgress: true}
	h.idempotencyMu.Unlock()

	// releaseKey removes the in-progress marker after a failure, so the
	// client can retry with the same key.
	releaseKey := func() {
		h.idempotencyMu.Lock()
		delete(h.idempotencyCache, cacheKey)
		h.idempotencyMu.Unlock()
	}

	// STEP 2: the slow work, with no lock held.
	result, err := h.orderService.CreateOrder(r.Context(), userID)
	if err != nil {
		releaseKey()
		handleError(w, r, err)
		return
	}

	// Order is created as "pending".
	// Actual payment confirmation happens later via Xsolla webhook.
	statusCode := http.StatusCreated

	responseBody, err := json.Marshal(map[string]any{
		"order":           result.Order,
		"xsolla_order_id": result.XsollaOrderID,
		"payment_url":     result.PaymentURL,
	})
	if err != nil {
		releaseKey()
		handleError(w, r, err)
		return
	}

	// STEP 3: store the finished response for replays.
	h.idempotencyMu.Lock()
	h.idempotencyCache[cacheKey] = &IdempotencyRecord{
		Response:   responseBody,
		StatusCode: statusCode,
		Expiry:     time.Now().Add(24 * time.Hour),
	}
	h.idempotencyMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(responseBody)
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
		writeError(
			w,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication is required",
		)
		return
	}

	limit, err :=
		intQuery(
			r,
			"limit",
			services.DefaultLimit,
		)

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_QUERY_PARAMETER",
			err.Error(),
		)
		return
	}

	offset, err :=
		intQuery(
			r,
			"offset",
			0,
		)

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_QUERY_PARAMETER",
			err.Error(),
		)
		return
	}

	result, err :=
		h.orderService.
			GetOrdersByUserIDOffset(
				r.Context(),
				userID,
				limit,
				offset,
			)

	if err != nil {
		handleError(w, r, err)
		return
	}

	response :=
		make(
			[]UserOrderResponse,
			0,
			len(result.Orders),
		)

	for _, order := range result.Orders {
		response = append(
			response,
			mapUserOrderResponse(
				order,
			),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		OrderOffsetResponse{
			Orders: response,
			Limit:  result.Limit,
			Offset: result.Offset,
		},
	)
}

func (h *Handler) GetUserOrdersCursor(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middlewares.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication is required",
		)
		return
	}

	cursor :=
		r.URL.Query().
			Get("cursor")

	limit, err :=
		intQuery(
			r,
			"limit",
			services.DefaultLimit,
		)

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_QUERY_PARAMETER",
			err.Error(),
		)
		return
	}

	result, err :=
		h.orderService.
			GetOrdersByUserIDCursor(
				r.Context(),
				userID,
				cursor,
				limit,
			)

	if err != nil {
		handleError(w, r, err)
		return
	}

	response :=
		make(
			[]UserOrderResponse,
			0,
			len(result.Orders),
		)

	for _, order := range result.Orders {
		response = append(
			response,
			mapUserOrderResponse(
				order,
			),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		OrderCursorResponse{
			Orders:     response,
			NextCursor: result.NextCursor,
			HasMore:    result.HasMore,
		},
	)
}

func mapUserOrderResponse(
	order models.UserOrder,
) UserOrderResponse {
	items :=
		make(
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

	return UserOrderResponse{
		ID:     order.ID,
		Items:  items,
		Total:  order.Total,
		Status: order.Status,
	}
}

// GetUserOrder godoc
// @Summary Get one of my orders
// @Description Returns an order of the logged-in user. Another user's order gives 404, the same as a missing one.
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Param order_id path int true "Order ID"
// @Success 200 {object} models.UserOrder
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /user/orders/{order_id} [get]
func (h *Handler) GetUserOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	orderID, err := strconv.Atoi(r.PathValue("order_id"))
	if err != nil || orderID <= 0 {
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND", "Order was not found")
		return
	}

	order, err := h.orderService.GetOrderForUser(r.Context(), orderID, userID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}
