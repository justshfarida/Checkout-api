package dtos

type CreateUserCartRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

type CreateOrderRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

type UpdateQuantityRequest struct {
	UserID   int `json:"user_id"`
	Quantity int `json:"quantity"`
}

type SignUpRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
