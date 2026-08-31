package handlers

import (
	"checkout-api/internal/controllers/pagination"
	"checkout-api/internal/models"
	"context"
	"net/http"
	"strconv"
)

type ItemsService interface {
	ListOffset(ctx context.Context, filter models.ItemFilter, page pagination.OffsetPage) (*pagination.PageResult[*models.Item], error)
	ListCursor(ctx context.Context, filter models.ItemFilter, page pagination.CursorPage) (*pagination.PageResult[*models.Item], error)
	GetItem(ctx context.Context, id int) (*models.Item, error)
}

type ItemsHandler struct {
	service ItemsService
}

func NewItemsHandler(service ItemsService) *ItemsHandler {
	return &ItemsHandler{service: service}
}

// GetItems lists items with filtering, sorting, and pagination.
// @Summary List items
// @Description Returns a paginated list of items. Pass "cursor" for cursor pagination, or "offset" for offset pagination.
// @Tags items
// @Produce json
// @Param category query string false "Filter by category"
// @Param sort query string false "Sort field: price or name"
// @Param limit query int false "Page size, max 100"
// @Param offset query int false "Offset (offset pagination)"
// @Param cursor query string false "Opaque cursor (cursor pagination)"
// @Success 200 {object} pagination.PageResult[models.Item]
// @Failure 400 {object} httperror.Envelope
// @Router /items [get]
func (handler *ItemsHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := models.ItemFilter{Category: query.Get("category"), SortBy: query.Get("sort")}
	ctx := r.Context()
	w.Header().Set("Cache-Control", "public, max-age=60")

	if query.Has("cursor") {
		page, err := pagination.ParseCursorPage(r)
		if err != nil {
			WriteValidationError(w, err.Error())
			return
		}
		result, err := handler.service.ListCursor(ctx, filter, page)
		if err != nil {
			WriteServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	page := pagination.ParseOffsetPage(r)
	result, err := handler.service.ListOffset(ctx, filter, page)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetItem fetches a single item by ID.
// @Summary Get an item
// @Tags items
// @Produce json
// @Param id path int true "Item ID"
// @Success 200 {object} models.Item
// @Failure 400 {object} httperror.Envelope
// @Failure 404 {object} httperror.Envelope
// @Router /items/{id} [get]
func (handler *ItemsHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	itemID, err := strconv.Atoi(idStr)
	if err != nil || itemID <= 0 {
		WriteValidationError(w, "invalid item id")
		return
	}

	item, err := handler.service.GetItem(r.Context(), itemID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}
