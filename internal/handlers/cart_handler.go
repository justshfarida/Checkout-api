package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/services"
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
	userID, ok := middlewares.UserIDFromContext(
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

	limit, err := intQuery(
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

	offset, err := intQuery(
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

	result, err := h.cartService.GetUserCartOffset(
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

	items := make(
		[]CartItemResponse,
		0,
		len(result.Items),
	)

	for _, item := range result.Items {
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
		CartOffsetResponse{
			UserID: userID,
			Items:  items,
			Limit:  result.Limit,
			Offset: result.Offset,
		},
	)
}

func (h *Handler) GetUserCartCursor(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := middlewares.UserIDFromContext(
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

	cursor := r.URL.Query().Get(
		"cursor",
	)

	limit, err := intQuery(
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

	result, err := h.cartService.GetUserCartCursor(
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

	items := make(
		[]CartItemResponse,
		0,
		len(result.Items),
	)

	for _, item := range result.Items {
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
		CartCursorResponse{
			UserID:     userID,
			Items:      items,
			NextCursor: result.NextCursor,
			HasMore:    result.HasMore,
		},
	)
}
