package handlers

import "time"

// PaymentResult represents a response from the payment provider.
type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type ItemResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

type LinteItemResponse struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"` // Price at time of adding
}

type OrderResponse struct {
	ID     int                 `json:"id"`
	UserID int                 `json:"user_id"`
	Items  []LinteItemResponse `json:"line_items"`
	Total  int                 `json:"total"`
	Status string              `json:"status"` // pending, paid, failed
}

type CartOffsetResponse struct {
	UserID int                `json:"user_id"`
	Items  []CartItemResponse `json:"items"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type CartCursorResponse struct {
	UserID     int                `json:"user_id"`
	Items      []CartItemResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
}

type CartItemResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	Quantity    int       `json:"quantity"`
}

type AuthResponse struct {
	JWT string `json:"jwt"`
}
type OrderItemResponse struct {
	ItemID   int    `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
}

type UserOrderResponse struct {
	ID     int                 `json:"id"`
	Items  []OrderItemResponse `json:"items"`
	Total  int                 `json:"total"`
	Status string              `json:"status"`
}
type OrderOffsetResponse struct {
	Orders []UserOrderResponse `json:"orders"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

type OrderCursorResponse struct {
	Orders     []UserOrderResponse `json:"orders"`
	NextCursor string              `json:"next_cursor,omitempty"`
	HasMore    bool                `json:"has_more"`
}
