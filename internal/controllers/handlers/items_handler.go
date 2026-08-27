package handlers

import (
	"checkout-api/internal/models"
	"context"
	"net/http"
	"strconv"
)

type ItemsService interface {
	GetItems(ctx context.Context, category, sortBy string) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
}

type ItemsHandler struct {
	service ItemsService
}

func NewItemsHandler(service ItemsService) *ItemsHandler {
	return &ItemsHandler{service: service}
}

func (handler *ItemsHandler) GetItems(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	category := query.Get("category")
	sortBy := query.Get("sort")

	items, err := handler.service.GetItems(request.Context(), category, sortBy)
	if err != nil {
		http.Error(writer, "failed to fetch items", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(writer, http.StatusOK, items)
}

func (handler *ItemsHandler) GetItem(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	id := query.Get("id")
	itemID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	item, err := handler.service.GetItem(request.Context(), itemID)
	if err != nil {
		http.Error(writer, http.StatusText(404), http.StatusNotFound)
		return
	}

	writeJSON(writer, http.StatusOK, item)
}
