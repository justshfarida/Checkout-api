package models

import "time"

type Item struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"` // Price in cents
	Stock       int    `json:"stock"`
	Category    string `json:"category"`
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
