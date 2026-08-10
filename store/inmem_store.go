package store

import (
	"checkout-api/models"
	"context"
	"fmt"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// InMemStore is an in-memory store for items, carts, and orders.
type InMemStore struct {
	items       map[int]*models.Item
	orders      map[int]*models.Order
	carts       map[int]*models.Cart
	idempotency map[string]*models.IdempotencyRecord
	users       map[int]*models.User
	nextOrderID int
	nextUserID  int
	sessions    map[string]*models.Session
}

// NewInMemStore creates a Store pre-loaded with seed data.
func NewInMemStore() *InMemStore {
	s := &InMemStore{
		items:       make(map[int]*models.Item),
		orders:      make(map[int]*models.Order),
		carts:       make(map[int]*models.Cart),
		idempotency: make(map[string]*models.IdempotencyRecord),
		users:       make(map[int]*models.User),
		nextOrderID: 1,
		nextUserID:  1,
		sessions:    make(map[string]*models.Session),
	}

	s.items[1] = &models.Item{ID: 1, Name: "Laptop", Description: "A fast laptop", Price: 120000, Stock: 10, Category: "electronics"}
	s.items[2] = &models.Item{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 2500, Stock: 50, Category: "electronics"}
	s.items[3] = &models.Item{ID: 3, Name: "Keyboard", Description: "Mechanical keyboard", Price: 8000, Stock: 30, Category: "electronics"}

	return s
}

// Items

func (s *InMemStore) GetItems(_ context.Context) ([]*models.Item, error) {
	items := make([]*models.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *InMemStore) GetItem(_ context.Context, id int) (*models.Item, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("item %d not found", id)
	}
	return item, nil
}

// Cart

func (s *InMemStore) CreateUserCart(_ context.Context, cart *models.Cart) error {
	cart.LastAccessed = time.Now()
	s.carts[cart.UserID] = cart
	return nil
}

func (s *InMemStore) GetUserCart(_ context.Context, userID int) (*models.Cart, error) {
	cart, ok := s.carts[userID]
	if !ok {
		return nil, nil
	}
	return cart, nil
}

func (s *InMemStore) DeleteUserCart(_ context.Context, userID int) error {
	delete(s.carts, userID)
	return nil
}

func (s *InMemStore) UpdateCartItem(_ context.Context, userID, itemID, quantity int) (bool, error) {
	cart, ok := s.carts[userID]
	if !ok {
		return false, nil
	}
	for i, item := range cart.Items {
		if item.ItemID == itemID {
			cart.Items[i].Quantity = quantity
			cart.LastAccessed = time.Now()
			return true, nil
		}
	}
	return false, nil
}

func (s *InMemStore) RemoveCartItem(_ context.Context, userID, itemID int) (bool, error) {
	cart, ok := s.carts[userID]
	if !ok {
		return false, nil
	}
	for i, item := range cart.Items {
		if item.ItemID == itemID {
			cart.Items = slices.Delete(cart.Items, i, i+1)
			if len(cart.Items) == 0 {
				delete(s.carts, userID)
			} else {
				cart.LastAccessed = time.Now()
			}
			return true, nil
		}
	}
	return false, nil
}

// Orders

func (s *InMemStore) CreateOrder(_ context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	order := &models.Order{
		ID:         s.nextOrderID,
		UserID:     userID,
		Items:      items,
		TotalPrice: total,
		Status:     status,
	}
	s.orders[order.ID] = order
	s.nextOrderID++
	return order, nil
}

func (s *InMemStore) GetOrdersByUserID(_ context.Context, userID int) ([]*models.Order, error) {
	orders := make([]*models.Order, 0)
	for _, order := range s.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

// Users

func (store *InMemStore) SignUp(_ context.Context, username string, passwordHash string) (*models.User, error) {
	for _, u := range store.users {
		if u.Username == username {
			return nil, fmt.Errorf("username %q already taken", username)
		}
	}

	user := &models.User{
		Username:     username,
		PasswordHash: passwordHash,
	}
	store.users[user.ID] = user
	store.nextUserID++
	return user, nil
}

func (s *InMemStore) Login(_ context.Context, username, password string) (*models.User, error) {
	for _, u := range s.users {
		if u.Username == username {
			if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
				return nil, fmt.Errorf("invalid username or password")
			}
			return u, nil
		}
	}
	return nil, fmt.Errorf("invalid username or password")
}

// Idempotency

func (s *InMemStore) GetIdempotencyRecord(_ context.Context, key string) (*models.IdempotencyRecord, error) {
	record, ok := s.idempotency[key]
	if !ok {
		return nil, nil
	}
	if time.Since(record.CreatedAt) > 30*time.Minute {
		delete(s.idempotency, key)
		return nil, nil
	}
	return record, nil
}

func (s *InMemStore) SaveIdempotencyRecord(_ context.Context, key, requestHash string, statusCode int, body any) error {
	s.idempotency[key] = &models.IdempotencyRecord{
		StatusCode:  statusCode,
		Body:        body,
		RequestHash: requestHash,
		CreatedAt:   time.Now(),
	}
	return nil
}
