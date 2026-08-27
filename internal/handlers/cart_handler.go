package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"checkout-api/internal/middlewares"
)

func (h *Handler) UpsertCartItem(
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

	itemIDStr :=
		r.PathValue("item_id")

	if itemIDStr == "" {
		http.Error(
			w,
			"missing item_id",
			http.StatusBadRequest,
		)
		return
	}

	itemID, err :=
		strconv.Atoi(itemIDStr)

	if err != nil {
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			ErrorMessageResponse{
				Message: "item_id must be integer",
			},
		)
		return
	}

	var req UpsertCartItemRequest

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

	if req.Quantity <= 0 {
		http.Error(
			w,
			"quantity must be greater than 0",
			http.StatusBadRequest,
		)
		return
	}

	err =
		h.cartService.UpsertCartItem(
			r.Context(),
			userID,
			itemID,
			req.Quantity,
		)

	if err != nil {
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func (h *Handler) RemoveCartItem(
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

	itemIDStr :=
		r.PathValue("item_id")

	if itemIDStr == "" {
		http.Error(
			w,
			"missing item_id",
			http.StatusBadRequest,
		)
		return
	}

	itemID, err :=
		strconv.Atoi(itemIDStr)

	if err != nil {
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			ErrorMessageResponse{
				Message: "item_id must be integer",
			},
		)
		return
	}

	err =
		h.cartService.RemoveCartItem(
			r.Context(),
			userID,
			itemID,
		)

	if err != nil {
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func (h *Handler) GetUserCart(
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

	cart, err :=
		h.cartService.GetUserCart(
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

	items :=
		make(
			[]CartItemResponse,
			0,
			len(cart),
		)

	for _, item := range cart {
		items = append(
			items,
			CartItemResponse{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Price:       item.Price,
				Stock:       item.Stock,
				CreatedAt:   item.CreatedAt,
				Quantity:    item.Quantity,
			},
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		CartResponse{
			UserID: userID,
			Items:  items,
		},
	)
}
