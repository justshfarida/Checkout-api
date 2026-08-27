package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/models"
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

	if err :=
		json.NewDecoder(
			r.Body,
		).Decode(&req); err != nil {

		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if len(req.LineItems) == 0 {
		http.Error(
			w,
			"items must not be empty",
			http.StatusBadRequest,
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

	orders, err :=
		h.orderService.
			GetOrdersByUserID(
				r.Context(),
				userID,
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
			len(orders),
		)

	for _, order := range orders {

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

		response = append(
			response,
			UserOrderResponse{
				ID:     order.ID,
				Items:  items,
				Total:  order.Total,
				Status: order.Status,
			},
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}
