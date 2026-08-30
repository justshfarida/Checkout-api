package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"checkout-api/internal/middlewares"
	"checkout-api/internal/services"
)

// UpsertCartItem godoc
// @Summary Add or update cart item
// @Description Adds an item to the current user's cart or updates its quantity.
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param item_id path int true "Item ID"
// @Param request body UpsertCartItemRequest true "Cart item quantity"
// @Success 204 "Cart item updated successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/cart/items/{item_id} [patch]
func (h *Handler) UpsertCartItem(
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

	itemIDStr :=
		r.PathValue("item_id")

	if itemIDStr == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_ITEM_ID",
			"item_id is required",
		)
		return
	}

	itemID, err :=
		strconv.Atoi(itemIDStr)

	if err != nil || itemID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_ITEM_ID",
			"item_id must be a positive integer",
		)
		return
	}

	var req UpsertCartItemRequest

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

	err =
		h.cartService.UpsertCartItem(
			r.Context(),
			userID,
			itemID,
			req.Quantity,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

// RemoveCartItem godoc
// @Summary Remove item from cart
// @Description Removes an item from the current user's cart.
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Param item_id path int true "Item ID"
// @Success 204 "Cart item removed successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/cart/items/{item_id} [delete]
func (h *Handler) RemoveCartItem(
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

	itemIDStr :=
		r.PathValue("item_id")

	if itemIDStr == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_ITEM_ID",
			"item_id is required",
		)
		return
	}

	itemID, err :=
		strconv.Atoi(itemIDStr)

	if err != nil || itemID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_ITEM_ID",
			"item_id must be a positive integer",
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
		handleError(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

// GetUserCart godoc
// @Summary Get current user's cart
// @Description Returns cart items using offset-based pagination.
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Maximum number of cart items to return" default(10)
// @Param offset query int false "Number of cart items to skip" default(0)
// @Success 200 {object} CartOffsetResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/cart [get]
func (h *Handler) GetUserCart(
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
		h.cartService.GetUserCartOffset(
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

	items :=
		make(
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
) { // GetUserCartCursor godoc
	// @Summary Get current user's cart using cursor pagination
	// @Description Returns cart items after the supplied cursor. Use next_cursor from the previous response to retrieve the next page.
	// @Tags Cart
	// @Produce json
	// @Security BearerAuth
	// @Param limit query int false "Maximum number of cart items to return" default(10)
	// @Param cursor query string false "Cursor returned by the previous page"
	// @Success 200 {object} CartCursorResponse
	// @Failure 400 {object} ErrorResponse
	// @Failure 401 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /user/cart/cursor [get]
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
		h.cartService.GetUserCartCursor(
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

	items :=
		make(
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
