package handlers

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/models"
	"checkout-api/internal/service"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

type CartService interface {
	CreateUserCartAndAddItems(ctx context.Context, userID int, items []service.CartItemInput) (*models.Cart, error)
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
	UpdateCartItem(ctx context.Context, userID, itemID, quantity int) (*models.Cart, error)
	RemoveCartItem(ctx context.Context, userID, itemID int) (*models.Cart, error)
}

type CartHandler struct {
	cartService CartService
}

func NewCartHandler(cartService CartService) *CartHandler {
	return &CartHandler{cartService: cartService}
}

// CreateUserCartAndAddItems creates a cart and adds items in one call.
// @Summary Create a cart and add items
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.CreateUserCartRequest true "Line items to add"
// @Success 201 {object} models.Cart
// @Failure 400 {object} httperror.Envelope
// @Failure 401 {object} httperror.Envelope
// @Router /user/cart [post]
func (handler *CartHandler) CreateUserCartAndAddItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteMethodNotAllowed(w)
		return
	}
	ctx := r.Context()

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		WriteUnauthorized(w, "unauthorized")
		return
	}

	var req dtos.CreateUserCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		WriteValidationError(w, "can't create empty cart")
		return
	}

	items := make([]service.CartItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, service.CartItemInput{ItemID: item.ItemID, Quantity: item.Quantity})
	}

	cart, err := handler.cartService.CreateUserCartAndAddItems(ctx, userID, items)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cart)
}

// GetUserCart returns the authenticated user's cart.
// @Summary Get the current user's cart
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Cart
// @Failure 401 {object} httperror.Envelope
// @Router /user/cart [get]
func (handler *CartHandler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w, "unauthorized")
		return
	}

	cart, err := handler.cartService.GetUserCart(r.Context(), userID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	if cart == nil {
		cart = &models.Cart{UserID: userID, Items: []models.LineItem{}}
	}
	writeJSON(w, http.StatusOK, cart)
}

// UpdateItemQuantity sets the quantity of an item already in the cart.
// @Summary Update a cart item's quantity
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Item ID"
// @Param request body dtos.UpdateQuantityRequest true "New quantity"
// @Success 200 {object} models.Cart
// @Failure 400 {object} httperror.Envelope
// @Failure 401 {object} httperror.Envelope
// @Router /user/cart/items/{id} [patch]
func (handler *CartHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		WriteValidationError(w, "invalid item id")
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w, "unauthorized")
		return
	}

	var req dtos.UpdateQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "invalid request body")
		return
	}
	if req.Quantity <= 0 {
		WriteValidationError(w, "quantity must be greater than 0")
		return
	}

	cart, err := handler.cartService.UpdateCartItem(r.Context(), userID, itemID, req.Quantity)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// DeleteItem removes an item from the cart.
// @Summary Remove an item from the cart
// @Tags cart
// @Security BearerAuth
// @Param id path int true "Item ID"
// @Success 200 {object} models.Cart
// @Success 204 "No Content"
// @Failure 400 {object} httperror.Envelope
// @Failure 401 {object} httperror.Envelope
// @Router /user/cart/items/{id} [delete]
func (handler *CartHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		WriteValidationError(w, "invalid item id")
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w, "unauthorized")
		return
	}

	cart, err := handler.cartService.RemoveCartItem(r.Context(), userID, itemID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	if cart == nil || len(cart.Items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
