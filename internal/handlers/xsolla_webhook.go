package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

type XsollaWebhook struct {
	NotificationType string `json:"notification_type"`
	User             struct {
		ID         string `json:"id"`
		ExternalId string `json:"external_id"`
	} `json:"user"`
	Order struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
		Mode   string `json:"mode"` //what is this
	} `json:"order"`
	Billing struct {
		Transaction struct {
			ID int64 `json:"id"`
		} `json:"transaction"`
	} `json:"billing"`
	Items            []XsollaWebhookItem `json:"items"`
	CustomParameters struct {
		OrderID string `json:"order_id"`
	} `json:"custom_parameters"`
}

type XsollaWebhookItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

func (h *Handler) XsollaWebhook(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body) //why readall

	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID REQUEST BODY", "failed to read request body")
		return
	}
	if !h.xsollaService.VerifyXsollaSignature(body, r.Header.Get("Authorization")) {
		slog.WarnContext(r.Context(), "Webhook rejected: invalid signature")
		writeError(w, http.StatusBadRequest, "INVALID SIGNATURE", "signature mismatch")
		return
	}
	var webhook XsollaWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		slog.WarnContext(r.Context(), "Webhook rejected: invalid JSON", "error", err.Error())
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid JSON body",
		)
		return
	}

	switch webhook.NotificationType {

	case "user_validation":
		h.handleXsollaUserValidation(
			w,
			r,
			webhook,
		)

	case "order_paid":
		h.handleXsollaOrderPaid(
			w,
			r,
			webhook,
		)

	case "order_canceled":
		h.handleXsollaOrderCanceled(
			w,
			r,
			webhook,
		)

	default:
		w.WriteHeader(http.StatusNoContent)

	}
}
func (h *Handler) handleXsollaUserValidation(w http.ResponseWriter, r *http.Request, webhook XsollaWebhook) {
	userId, err := strconv.Atoi(webhook.User.ID)
	if err != nil {
		writeError(
			w, http.StatusBadRequest, "INVALID_USER", "invalid user id")
		return
	}
	result, err := h.authService.UserExists(r.Context(), userId)
	if err != nil {
		slog.ErrorContext(r.Context(), "User validation failed",
			"error", err.Error(),
			"user_id", userId,
		)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to validate user")
		return
	}
	if !result {
		slog.WarnContext(r.Context(), "User validation: user not found", "user_id", userId)
		writeError(w, http.StatusBadRequest, "INVALID_USER", "user not found")
		return
	}
	w.WriteHeader(http.StatusOK)

}

func (h *Handler) handleXsollaOrderPaid(
	w http.ResponseWriter,
	r *http.Request,
	webhook XsollaWebhook,
) {
	// TODO:
	transactionID := int64(webhook.Billing.Transaction.ID)
	orderID, err := strconv.Atoi(webhook.CustomParameters.OrderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ORDER_ID", "invalid order id")
		return
	}

	err = h.orderService.ProcessXsollaPayment(
		r.Context(),
		orderID,
		transactionID,
	)
	if err != nil {
		slog.ErrorContext(r.Context(), "Payment processing failed",
			"error", err.Error(),
			"order_id", orderID,
			"transaction_id", transactionID,
		)

		writeError(
			w,
			http.StatusInternalServerError,
			"PAYMENT_PROCESSING_FAILED",
			"failed to process payment",
		)
		return
	}

	slog.InfoContext(r.Context(), "Payment processed",
		"order_id", orderID,
		"transaction_id", transactionID,
	)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleXsollaOrderCanceled(
	w http.ResponseWriter,
	r *http.Request,
	webhook XsollaWebhook,
) {
	orderID, err := strconv.Atoi(
		webhook.CustomParameters.OrderID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_ORDER_ID",
			"invalid order id",
		)
		return
	}

	err = h.orderService.UpdateOrderStatus(
		r.Context(),
		orderID,
		"canceled",
	)
	if err != nil {
		slog.ErrorContext(r.Context(), "Order cancel failed",
			"error", err.Error(),
			"order_id", orderID,
		)
		writeError(
			w,
			http.StatusInternalServerError,
			"ORDER_UPDATE_FAILED",
			"failed to cancel order",
		)
		return
	}

	slog.InfoContext(r.Context(), "Order canceled", "order_id", orderID)

	w.WriteHeader(http.StatusOK)
}
