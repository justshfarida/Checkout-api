package handlers

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/controllers"
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type CartService interface {
	CreateUserCartAndAddItems(ctx context.Context, cart *models.Cart) error
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
	UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (bool, error)
	RemoveCartItem(ctx context.Context, userID, itemID int) (bool, error)
}

type CartHandler struct {
	service CartService
}

func NewCartHandler(service CartService) *CartHandler {
	return &CartHandler{service: service}
}

func (handler *CartHandler) CreateUserCartAndAddItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req dtos.CreateUserCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "can't create empty cart", http.StatusBadRequest)
		return
	}

	orderItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem, err := handler.service.GetItem(ctx, item.ItemID)
		if err != nil {
			http.Error(w, fmt.Sprintf("item %d not found", item.ItemID), http.StatusBadRequest)
			return
		}
		orderItems = append(orderItems, models.LineItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	userCart := &models.Cart{
		ID:     fmt.Sprintf("cart_%d", userID),
		UserID: userID, // use the authenticated user, not the request body — closes the earlier-flagged trust gap too
		Items:  orderItems,
	}

	if err := handler.service.CreateUserCartAndAddItems(ctx, userCart); err != nil {
		http.Error(w, "failed to create cart", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, userCart)
}

func (handler *CartHandler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	cart, err := handler.service.GetUserCart(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to fetch cart", http.StatusInternalServerError)
		return
	}
	if cart == nil {
		cart = &models.Cart{ID: "", UserID: userID, Items: []models.LineItem{}}
	}
	writeJSON(w, http.StatusOK, cart)
}

func (handler *CartHandler) UpdateItemQuantity(writer http.ResponseWriter, request *http.Request) {
	idStr := request.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	// BUSINESS

	var req controllers.updateQuantityRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		http.Error(writer, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(writer, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}
	//
	ctx := request.Context()
	found, err := handler.service.UpdateCartItem(ctx, userID, id, req.Quantity)
	if err != nil {
		http.Error(writer, "failed to update cart item", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	cart, err := handler.service.GetUserCart(ctx, userID)
	if err != nil {
		http.Error(writer, "failed to fetch updated cart", http.StatusInternalServerError)
		return
	}
	writeJSON(writer, http.StatusOK, cart)
}

func (handler *CartHandler) DeleteItem(writer http.ResponseWriter, request *http.Request) {
	idStr := request.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(400), http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := request.Context()
	found, err := handler.service.RemoveCartItem(ctx, userID, id)
	if err != nil {
		http.Error(writer, "failed to remove cart item", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	cart, err := handler.service.GetUserCart(ctx, userID)
	if err != nil {
		http.Error(writer, "failed to fetch cart", http.StatusInternalServerError)
		return
	}
	if cart == nil || len(cart.Items) == 0 {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(writer, http.StatusOK, cart)
}
