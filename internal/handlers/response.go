package handlers

import "time"

type CartOffsetResponse struct {
	Items  []CartItemResponse `json:"items"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
	Total  int                `json:"total"`
}

type CartCursorResponse struct {
	Items      []CartItemResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
	Total      int                `json:"total"`
}

type CartItemResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	Quantity    int       `json:"quantity"`
	// ItemType lets the frontend decide whether stock applies (only "merch").
	ItemType     string `json:"item_type"`
	CreditAmount *int   `json:"credit_amount"`
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
