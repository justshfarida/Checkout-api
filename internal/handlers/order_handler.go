package handlers

import (
	"encoding/json"
	"net/http"
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

	if record, exists :=
		h.idempotencyCache[idempotencyKey]; exists {

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
			idempotencyKey,
		)
	}

	var req CreateOrderRequest

	dec :=
		json.NewDecoder(
			r.Body,
		)

	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Invalid request body",
		)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"VALIDATION_FAILED",
			"Request validation failed",
			validationDetails(err)...,
		)
		return
	}

	items :=
		make(
			[]models.LineItem,
			0,
			len(req.LineItems),
		)

	for _, i := range req.LineItems {
		items = append(
			items,
			models.LineItem{
				ItemID:   i.ItemID,
				Quantity: i.Quantity,
				Price:    i.Price,
			},
		)
	}

	result, err :=
		h.orderService.CreateOrder(
			r.Context(),
			userID,
			items,
			req.Total,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	statusCode :=
		http.StatusCreated

	if !result.Payment.Success {
		statusCode =
			http.StatusPaymentRequired
	}

	responseData :=
		map[string]any{
			"order":   result.Order,
			"payment": result.Payment,
		}

	responseBody, err :=
		json.Marshal(
			responseData,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	h.idempotencyCache[idempotencyKey] =
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
		handleError(
			w,
			err,
		)
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
		handleError(
			w,
			err,
		)
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
