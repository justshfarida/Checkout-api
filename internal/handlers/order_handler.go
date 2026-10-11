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

	// Hold the lock for the whole request so concurrent retries with the
	// same key can't create duplicate orders.
	h.idempotencyMu.Lock()
	defer h.idempotencyMu.Unlock()

	// Return cached response for the same request.
	if record, exists :=
		h.idempotencyCache[cacheKey]; exists {

		if time.Now().Before(record.Expiry) {
			w.Header().
				Set(
					"Content-Type",
					"application/json",
				)

			w.WriteHeader(
				record.StatusCode,
			)

			_, _ =
				w.Write(
					record.Response,
				)

			return
		}

		delete(
			h.idempotencyCache,
			cacheKey,
		)
	}

	result, err :=
		h.orderService.CreateOrder(
			r.Context(),
			userID,
		)

	if err != nil {
		handleError(w, r, err)
		return
	}

	// Order is created as "pending".
	// Actual payment confirmation happens later via Xsolla webhook.
	statusCode := http.StatusCreated

	responseData :=
		map[string]any{
			"order":           result.Order,
			"xsolla_order_id": result.XsollaOrderID,
			"payment_url":     result.PaymentURL,
		}

	responseBody, err :=
		json.Marshal(
			responseData,
		)

	if err != nil {
		handleError(w, r, err)
		return
	}

	h.idempotencyCache[cacheKey] =
		&IdempotencyRecord{
			Response:   responseBody,
			StatusCode: statusCode,
			Expiry: time.Now().
				Add(
					24 * time.Hour,
				),
		}

	w.Header().
		Set(
			"Content-Type",
			"application/json",
		)

	w.WriteHeader(
		statusCode,
	)

	_, _ =
		w.Write(
			responseBody,
		)
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
