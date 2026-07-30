package store

import (
	"checkout-api/models"
	"sort"
)

// Implement in-memory Store with NewStore() constructor.
// Seed data: Laptop (120000), Mouse (2500), Keyboard (8000) — prices in cents.
// Methods: GetItems(), GetItem(id int), CreateOrder(userID int, items []models.LineItem, total int, status string)

type Store struct {
	Items       map[int]*models.Item
	Orders      map[int]*models.Order
	NextOrderId int
}

func NewStore() *Store {
	store := &Store{
		Items:       make(map[int]*models.Item),
		Orders:      make(map[int]*models.Order),
		NextOrderId: 1,
	}

	store.Items[1] = &models.Item{
		ID:          1,
		Name:        "Laptop",
		Description: "Laptop",
		Price:       120000,
		StockSize:   100,
		Category:    "electronics",
	}
	store.Items[2] = &models.Item{
		ID:          2,
		Name:        "Mouse",
		Description: "Mouse",
		Price:       2500,
		StockSize:   100,
		Category:    "electronics",
	}
	store.Items[3] = &models.Item{
		ID:          3,
		Name:        "Keyboard",
		Description: "Keyboard",
		Price:       8000,
		StockSize:   100,
		Category:    "electronics",
	}
	store.Items[4] = &models.Item{
		ID:          4,
		Name:        "Monitor",
		Description: "4K UHD Display",
		Price:       35000,
		StockSize:   50,
		Category:    "electronics",
	}
	store.Items[5] = &models.Item{
		ID:          5,
		Name:        "Desk Chair",
		Description: "Ergonomic Office Chair",
		Price:       15000,
		StockSize:   20,
		Category:    "furniture",
	}
	store.Items[6] = &models.Item{
		ID:          6,
		Name:        "Standing Desk",
		Description: "Adjustable Height Desk",
		Price:       45000,
		StockSize:   15,
		Category:    "furniture",
	}
	store.Items[7] = &models.Item{
		ID:          7,
		Name:        "Go Programming Book",
		Description: "Learn Go from Scratch",
		Price:       3000,
		StockSize:   200,
		Category:    "books",
	}
	store.Items[8] = &models.Item{
		ID:          8,
		Name:        "Coffee Mug",
		Description: "Ceramic Mug 350ml",
		Price:       1200,
		StockSize:   150,
		Category:    "kitchen",
	}

	return store
}

func (store *Store) GetItems(filter *models.ItemFilter) []*models.Item {

	items := make([]*models.Item, 0, len(store.Items))
	for _, item := range store.Items {
		items = append(items, item)
	}
	if filter == nil {
		return items
	}
	if filter.Category != "" {
		var filtered []*models.Item
		for _, item := range items {
			if item.Category == filter.Category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if filter.SortBy != "" {
		switch result := filter.SortBy; result {
		case "price":
			sort.Slice(items, func(i int, j int) bool {
				return items[i].Price < items[j].Price
				// "Is left smaller than right? Yes (true) -> Keep left BEFORE right."
			})
		}

	}

	return items
}
func (store *Store) GetItem(id int) *models.Item {
	for _, item := range store.Items {

		if item.ID == id {
			return item
		}
	}
	return nil
}
func (store *Store) CreateOrder(orderRequest *models.CreateOrderRequest) *models.Order {

	total := 0
	for _, item := range orderRequest.Items {
		total += item.Price * item.Quantity
	}
	store.Orders[store.NextOrderId] = &models.Order{
		ID:         store.NextOrderId,
		UserID:     orderRequest.UserID,
		Items:      orderRequest.Items,
		TotalPrice: total,
		// Status:     status,
	}
	store.NextOrderId++
	return store.Orders[store.NextOrderId-1]
}
func (store *Store) GetOrdersByUserID(userID int) []*models.Order {
	var userOrders []*models.Order
	for _, order := range store.Orders {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}
	return userOrders
}
