package handlers

import (
	"checkout-api/internal/services"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
		h.itemService.GetItemsOffset(
			r.Context(),
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
		http.Error(
			w,
			"Not Found",
			http.StatusNotFound,
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
				Message: "item_id must be an integer",
			},
		)
		return
	}

	item, err :=
		h.itemService.GetItem(
			r.Context(),
			itemID,
		)

	if err != nil {
		fmt.Printf(
			"failed to get item: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	if item == nil {
		writeJSON(
			w,
			http.StatusNotFound,
			nil,
		)
		return
	}

	writeJSON(

		w,
		http.StatusOK,
		item,
	)
}
func (h *Handler) GetItemsCursor(w http.ResponseWriter, r *http.Request) {
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
		h.itemService.GetItemsCursor(
			r.Context(),
			rawCursor,
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

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}
