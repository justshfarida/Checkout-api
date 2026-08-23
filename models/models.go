package models

import "time"

type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
}

type LineItem struct {
	ItemID   int    `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
}

type Order struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id"`
	Items      []LineItem `json:"items"`
	TotalPrice int        `json:"total"`
	Status     string     `json:"status"`
}

type Cart struct {
	ID           string     `json:"id"`
	UserID       int        `json:"user_id"`
	Items        []LineItem `json:"items"`
	LastAccessed time.Time  `json:"-"`
}

type IdempotencyRecord struct {
	StatusCode  int
	Body        any
	RequestHash string
	CreatedAt   time.Time
}

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	Token     string    `json:"-"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RefreshToken struct {
	Token     string    `json:"-"`
	UserID    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
