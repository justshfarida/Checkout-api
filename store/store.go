package store

import (
	"checkout-api/models"
	"fmt"
	"slices"
	"time"
)

// Implement in-memory Store with NewStore() constructor.
// Seed data: Laptop (120000), Mouse (2500), Keyboard (8000) — prices in cents.
// Methods: GetItems(), GetItem(id int), CreateOrder(userID int, items []models.LineItem, total int, status string)

// Store is an in-memory store for items and orders.
type Store struct {
	Items       map[int]*models.Item
	Orders      map[int]*models.Order
	nextOrderID int
	Carts       map[int]*models.Cart
	Idempotency map[string]*models.IdempotencyRecord
	Users       map[int]*models.User
	NextUserID  int
	Sessions    map[int]*models.Session
}

// NewStore creates a Store pre-loaded with seed data.
func NewStore() *Store {
	store := &Store{
		Items:       make(map[int]*models.Item),
		Orders:      make(map[int]*models.Order),
		nextOrderID: 1,
		Carts:       make(map[int]*models.Cart),
		Idempotency: make(map[string]*models.IdempotencyRecord),
		Users:       make(map[int]*models.User),
		NextUserID:  1,
		Sessions:    make(map[int]*models.Session),
	}

	store.Items[1] = &models.Item{ID: 1, Name: "Laptop", Description: "A fast laptop", Price: 120000, Stock: 10, Category: "electronics"}
	store.Items[2] = &models.Item{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 2500, Stock: 50, Category: "electronics"}
	store.Items[3] = &models.Item{ID: 3, Name: "Keyboard", Description: "Mechanical keyboard", Price: 8000, Stock: 30, Category: "electronics"}
	store.Items[4] = &models.Item{ID: 4, Name: "Book", Description: "Mechanical keyboard", Price: 20, Stock: 10, Category: "education"}

	store.Orders[1] = &models.Order{
		ID:     1,
		UserID: 1,
		Items: []models.LineItem{
			{ItemID: 1, Quantity: 3, Price: 2500},
			{ItemID: 2, Quantity: 1, Price: 1000},
		},
		TotalPrice: 8500,
		Status:     "active",
	}
	store.Orders[2] = &models.Order{
		ID:     2,
		UserID: 2,
		Items: []models.LineItem{
			{ItemID: 1, Quantity: 2, Price: 500},
			{ItemID: 2, Quantity: 5, Price: 1000},
		},
		TotalPrice: 6000,
		Status:     "active",
	}
	store.Orders[3] = &models.Order{
		ID:     3,
		UserID: 2,
		Items: []models.LineItem{
			{ItemID: 1, Quantity: 10, Price: 600},
			{ItemID: 2, Quantity: 2, Price: 4000},
		},
		TotalPrice: 14000,
		Status:     "active",
	}

	return store
}

// Items

func (store *Store) GetItems() []*models.Item {
	items := make([]*models.Item, 0, len(store.Items))
	for _, item := range store.Items {
		items = append(items, item)
	}
	return items
}

func (store *Store) GetItem(id int) *models.Item {
	return store.Items[id]
}

// Cart

func (store *Store) CreateUserCartAndAddItems(cart *models.Cart) {
	store.Carts[cart.UserID] = cart
}

func (store *Store) GetUserCart(userID int) *models.Cart {
	if c, ok := store.Carts[userID]; ok {
		return c
	}
	return nil
}

func (store *Store) UpdateItemQuantity(userID int, itemID int, quantity int) *models.Cart {
	cart, ok := store.Carts[userID]
	if !ok {
		return nil
	}

	for i, item := range cart.Items {
		if item.ItemID == itemID {
			cart.Items[i].Quantity = quantity
			return cart
		}
	}
	return nil
}

func (store *Store) DeleteItem(userID int, itemID int) (*models.Cart, bool, bool) {
	cart, ok := store.Carts[userID]
	if !ok {
		return nil, false, false
	}
	for i, item := range cart.Items {
		if item.ItemID == itemID {
			cart.Items = slices.Delete(cart.Items, i, i+1)
			if len(cart.Items) == 0 {
				delete(store.Carts, userID)
				return nil, true, true
			}
			return cart, false, true
		}
	}
	return nil, false, false
}

func (store *Store) DeleteCart(userID int) {
	delete(store.Carts, userID)
}

// Orders

func (store *Store) CreateOrder(userID int, items []models.LineItem, total int, status string) *models.Order {
	order := &models.Order{
		ID:         store.nextOrderID,
		UserID:     userID,
		Items:      items,
		TotalPrice: total,
		Status:     status,
	}
	store.Orders[order.ID] = order
	store.nextOrderID++
	return order
}

func (store *Store) GetOrdersByUserID(userID int) []*models.Order {
	orders := make([]*models.Order, 0, len(store.Orders))
	for _, order := range store.Orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders
}

// Users

func (store *Store) SignUp(username string, password string) (*models.User, error) {
	user := &models.User{
		Username:     username,
		PasswordHash: password,
	}
	store.Users[user.ID] = user
	store.NextUserID++
	return user, nil
}

func (store *Store) Login(username string, password string) {
	for _, u := range store.Users {
		if u.PasswordHash != password {
			fmt.Errorf("password incorrect for %q", username)
		}
	}

	fmt.Printf("You are logged in, %s\n", username)
}

// Idempotency

func (store *Store) GetIdempotencyRecord(key string) *models.IdempotencyRecord {
	record, ok := store.Idempotency[key]
	if !ok {
		return nil
	}
	if time.Since(record.CreatedAt) > 30*time.Minute {
		delete(store.Idempotency, key)
		return nil
	}
	return record
}

func (store *Store) SaveIdempotencyRecord(key, requestHash string, statusCode int, body any) {
	store.Idempotency[key] = &models.IdempotencyRecord{
		StatusCode:  statusCode,
		Body:        body,
		RequestHash: requestHash,
		CreatedAt:   time.Now(),
	}
}
