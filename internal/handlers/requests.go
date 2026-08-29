package handlers

type UpsertCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

// o is considered missing
type CreateOrderRequest struct {
	LineItems []LineItemRequest `json:"line_items" validate:"required,min=1,dive"`
	Total     int               `json:"total" validate:"required,gt=0"`
}

// dive means validate each LineItemRequest
type LineItemRequest struct {
	ItemID   int `json:"item_id" validate:"required,gt=0"`
	Quantity int `json:"quantity" validate:"required,gt=0"`
	Price    int `json:"price" validate:"required,gt=0"`
}

type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=25"`
}
