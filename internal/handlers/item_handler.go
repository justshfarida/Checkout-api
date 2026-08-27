package handlers

import (
	"fmt"
	"net/http"
	"strconv"
)

func (h *Handler) GetItems(
	w http.ResponseWriter,
	r *http.Request,
) {
	items, err :=
		h.itemService.GetItems(
			r.Context(),
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
		items,
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
