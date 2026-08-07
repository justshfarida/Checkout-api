package models

import "time"

type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

type LineItem struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"`
}

type Order struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id"`
	Items      []LineItem `json:"items"`
	TotalPrice int        `json:"total"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Cart struct {
	ID     string     `json:"id"`
	UserID int        `json:"user_id"`
	Items  []LineItem `json:"items"`
}
type CartItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	Quantity    int       `json:"quantity"`
}

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Hash  []byte
}
