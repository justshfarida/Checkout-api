package handlers

import (
	"net/http"
	"strconv"

	"checkout-api/internal/services"
)

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
