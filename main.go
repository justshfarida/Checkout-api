package main

import (
	"log"
	"net/http"

	"checkout-api/handlers"
	"checkout-api/store"
)

func main() {
	store := store.NewStore()
	handler := handlers.NewHandler(store)

	http.HandleFunc("/user/cart",
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.GetUserCart(w, r)
			case http.MethodPost:
				handler.CreateUserCartAndAddItems(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	http.HandleFunc("GET /items", handler.GetItems)
	http.HandleFunc("GET /items/", handler.GetItem)
	http.HandleFunc("POST /orders", handler.CreateOrder)
	http.HandleFunc("GET /users/{user_id}/orders", handler.GetOrdersByUserID)
	http.HandleFunc("PATCH /user/cart/items/{id}", handler.UpdateItemQuantity)
	http.HandleFunc("DELETE /user/cart/items/{id}", handler.DeleteItem)
	http.HandleFunc("GET /users", handler.GetItems)
	// http.HandleFunc("POST /user/orders", handler.PLaceOrder)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
