package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/models"
	"checkout-api/internal/services"
)

func mockProcessPayment(
	amount int,
) PaymentResult {
	amount /= 100

	if amount > 0 &&
		amount < 40 {

		return PaymentResult{
			Success: true,
			TransactionID: fmt.Sprintf(
				"txn_%d",
				time.Now().UnixNano(),
			),
		}
	}

	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}

func (h *Handler) CreateOrder(
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

	idempotencyKey :=
		r.Header.Get(
			"Idempotency-Key",
		)

	if idempotencyKey == "" {
		http.Error(
			w,
			"Idempotency-Key header is required",
			http.StatusBadRequest,
		)
		return
	}

	if record, exists := h.idempotencyCache[idempotencyKey]; exists {

		if time.Now().
			Before(record.Expiry) {

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

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			ErrorMessageResponse{
				Message: err.Error(),
			},
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

	order, err :=
		h.orderService.CreateOrder(
			r.Context(),
			userID,
			items,
			req.Total,
			"pending",
		)

	if err != nil {
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	paymentResult :=
		mockProcessPayment(
			req.Total,
		)

	status := "paid"

	if !paymentResult.Success {
		status = "failed"
	}

	err =
		h.orderService.
			UpdateOrderStatus(
				r.Context(),
				order.ID,
				status,
			)

	if err != nil {
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	order.Status = status

	responseData :=
		map[string]any{
			"order":   order,
			"payment": paymentResult,
		}

	statusCode :=
		http.StatusCreated

	if !paymentResult.Success {
		statusCode =
			http.StatusPaymentRequired
	}

	responseBody, _ :=
		json.Marshal(
			responseData,
		)

	h.idempotencyCache[idempotencyKey] = &IdempotencyRecord{
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
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
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
		writeJSON(
			w,
			http.StatusBadRequest,
			ErrorMessageResponse{
				Message: err.Error(),
			},
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
		writeJSON(
			w,
			http.StatusBadRequest,
			ErrorMessageResponse{
				Message: err.Error(),
			},
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
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
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
		writeJSON(
			w,
			http.StatusBadRequest,
			ErrorMessageResponse{
				Message: err.Error(),
			},
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
		if errors.Is(
			err,
			services.ErrInvalidCursor,
		) {
			writeJSON(
				w,
				http.StatusBadRequest,
				ErrorMessageResponse{
					Message: "invalid cursor",
				},
			)
			return
		}

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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
