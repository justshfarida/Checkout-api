package handlers

import (
	"checkout-api/models"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Implement HTTP handlers:
// - NewHandler(s *store.Store) — constructor
// - GetItems: GET /items → return all items as JSON
// - CreateOrder: POST /orders → decode request, validate items, calculate total, mock payment, create order

type ItemStore interface {
	GetItems(filter *models.ItemFilter) []*models.Item
	GetItem(id int) *models.Item
	CreateOrder(orderRequest *models.CreateOrderRequest) *models.Order
	GetOrdersByUserID(userID int) []*models.Order
}

type Handler struct {
	itemStore ItemStore // Store interface to access items and orders. we keep interface instead of concrete type to make it easier to test
}

func NewHandler(itemStore ItemStore) *Handler {
	return &Handler{
		itemStore: itemStore,
	}
}
func (handler *Handler) GetItems(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}
	category := request.URL.Query().Get("category")
	sort := request.URL.Query().Get("sort")

	itemFilter := &models.ItemFilter{Category: category, SortBy: sort}
	items := handler.itemStore.GetItems(itemFilter)
	writeJson(writer, http.StatusOK, items)
}
func (handler *Handler) GetItemByID(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}
	path := request.URL.Path

	idStr := strings.TrimPrefix(path, "/items/")
	id, err := strconv.Atoi(idStr)
	fmt.Printf("Item ID: %d\n", id)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	item := handler.itemStore.GetItem(id)
	if item == nil {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	writeJson(writer, http.StatusOK, item)

}
func ProcessPayment() error {
	delayTime := time.Millisecond * 100
	maxTries := 3
	for try := 1; try <= maxTries; try++ {
		val := rand.Intn(2)
		if val == 1 {
			return nil
		}
		time.Sleep(delayTime)
		delayTime *= 2
	}
	return fmt.Errorf("Payment Failed")
}
func (handler *Handler) CreateOrder(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}
	order := &models.CreateOrderRequest{}
	response := parseJson(request, order)
	if response != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	for _, item := range order.Items {
		lineItem := handler.itemStore.GetItem(item.ItemID)
		if lineItem == nil {
			http.Error(writer, "not found", http.StatusBadRequest)
			return
		}
	}
	// item := handler.itemStore.GetItem(order.Items[0].ItemID)
	// if item == nil {
	// 	http.Error(writer, "not found", http.StatusBadRequest)
	// 	return
	// }
	if err := ProcessPayment(); err != nil {
		http.Error(writer, err.Error(), http.StatusPaymentRequired) // Returns 402 if payment fails after retries
		return
	}
	handler.itemStore.CreateOrder(order)
	http.Error(writer, `"status":"paid"`, http.StatusCreated)

}
func (handler *Handler) GetOrdersByUserID(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	path := strings.TrimPrefix(request.URL.Path, "/users/") // Result: "452/orders"
	idStr := strings.TrimSuffix(path, "/orders")            // Result: "452"
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	orders := handler.itemStore.GetOrdersByUserID((id))
	if orders == nil {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	writeJson(writer, http.StatusOK, orders)

}
func writeJson(writer http.ResponseWriter, statusCode int, data any) {
	writer.Header().Add("Content-Type", "application/json")
	writer.Header().Add("Cache-Control", "private")
	writer.WriteHeader(statusCode)
	json.NewEncoder(writer).Encode(data)
}
func parseJson(request *http.Request, v any) error {
	if request.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	return json.NewDecoder(request.Body).Decode(v) //returns nil if successful, otherwise returns an error
	//it decodes the JSON data from the request body into the variable v
}
