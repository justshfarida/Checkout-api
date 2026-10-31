package handlers

type UpsertCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=25"`
}
