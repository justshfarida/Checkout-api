package handlers

import (
	"net/http"
	"strconv"

	"checkout-api/internal/services"
)

// GetItems godoc
// @Summary List items
// @Description Returns items using offset-based pagination.
// @Tags Items
// @Produce json
// @Param limit query int false "Maximum number of items to return" default(10)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} services.ItemOffsetResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /items [get]
func (h *Handler) GetItems(
	w http.ResponseWriter,
	r *http.Request,
) {
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
		h.itemService.GetItemsOffset(
			r.Context(),
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

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

// GetItemByID godoc
// @Summary Get item by ID
// @Description Returns a single item by its ID.
// @Tags Items
// @Produce json
// @Param item_id path int true "Item ID"
// @Success 200 {object} models.Item
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /items/{item_id} [get]
func (h *Handler) GetItemByID(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	item, err :=
		h.itemService.GetItem(
			r.Context(),
			itemID,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		item,
	)
}

// GetItemsCursor godoc
// @Summary List items using cursor pagination
// @Description Returns items after the supplied cursor. Use next_cursor from the previous response to retrieve the next page.
// @Tags Items
// @Produce json
// @Param limit query int false "Maximum number of items to return" default(10)
// @Param cursor query string false "Cursor returned by the previous page"
// @Success 200 {object} services.ItemCursorResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /items/cursor [get]
func (h *Handler) GetItemsCursor(
	w http.ResponseWriter,
	r *http.Request,
) {
	rawCursor :=
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
		h.itemService.GetItemsCursor(
			r.Context(),
			rawCursor,
			limit,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}
