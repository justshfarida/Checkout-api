package handlers

type UpsertCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

// o is considered missing
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
