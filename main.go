package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"checkout-api/handlers"
	"checkout-api/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	// conn, err := pgx.Connect(ctx, )
	// if err != nil {
	// 	panic(err)
	// }

	// err = conn.Ping(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("successfully connected to db")
	conn, err := pgxpool.New(ctx, "postgresql://postgres:mysecretpassword@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if err := conn.Ping(ctx); err != nil {
		panic(err)
	}
	// s := store.NewInMemStore()
	postgresStore := store.NewPostgresStore(conn)
	h := handlers.NewHandler(postgresStore)
	// fmt.Println("===== WITHOUT LOCK =====")
	// store.SimulateWithoutLock(postgresStore)
	fmt.Println("===== WITH LOCK =====")
	store.SimulateWithLock(postgresStore)
	http.HandleFunc("/user/cart/items/",
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPatch:
				h.UpdateCartItem(w, r)
			case http.MethodDelete:
				h.RemoveCartItem(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	http.HandleFunc("/user/cart",
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				h.GetUserCart(w, r)
			case http.MethodPost:
				h.CreateUserCartAndAddItems(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	http.HandleFunc("/user/orders", h.CreateOrderFromCart)

	http.HandleFunc("/items", h.GetItems)
	http.HandleFunc("/items/", h.GetItemByID)
	http.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.SignUp(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Server starting on :8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
