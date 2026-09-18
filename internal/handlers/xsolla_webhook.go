package handlers

import (
	"encoding/json"
	"io"
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
			ID int `json:"id"`
		} `json:"transaction"`
	} `json:"billing"`
	Items []XsollaWebhookItem `json:"items"`
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
	if !h.xsollaService.VerifyXsollaSignature(body,r.Header.Get("Authorization") ){
		writeError(w, http.StatusBadRequest, "INVALID SIGNATURE", "signature mismatch")
		return
	}
	var webhook XsollaWebhook
	if err:= json.Unmarshal(body, &webhook); err!=nil{
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid JSON body",
		)
		return
	}


	switch webhook.NotificationType{

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
func (h *Handler) handleXsollaUserValidation(w http.ResponseWriter, r *http.Request,webhook XsollaWebhook){
	// TODO: check that the user exists.
	userId, err:=strconv.Atoi(webhook.User.ID)
	if err!=nil{
		writeError(
			w, http.StatusBadRequest, "INVALID_USER", "invalid user id", 
		)
	}
	result, err:=h.authService.UserExists(r.Context(), userId);
	if err!=nil{
		writeError(w, http.StatusNotFound, "INTERNAL_ERROR", "failed to validate user")
	}
	if !result{
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
	// transactionID := webhook.Billing.Transaction.ID
	// orderID := webhook.CustomParameters.OrderID
	//
	// check transaction idempotency
	// mark order paid
	// grant goods

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleXsollaOrderCanceled(
	w http.ResponseWriter,
	r *http.Request,
	webhook XsollaWebhook,
) {
	// TODO: cancellation logic.
	w.WriteHeader(http.StatusNoContent)
}