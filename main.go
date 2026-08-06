package main

import (
	"checkout-api/handlers"
	"checkout-api/store"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found; using system environment variables")
	}

	databaseURL := mustGetEnv("DATABASE_URL")
	jwtSecret := mustGetEnv("JWT_SECRET")

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		panic(err)
	}

	err = conn.Ping(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("successfully connected to db")

	// s := store.NewInMemStore()
	postgresStore := store.NewPostgresStore(conn)
	h := handlers.NewHandler(postgresStore, []byte(jwtSecret))

	// cart
	http.HandleFunc("GET /user/cart", h.GetUserCart)
	http.HandleFunc("PATCH /user/cart/items/{item_id}", h.UpsertCartItem)
	http.HandleFunc("DELETE /user/cart/items/{item_id}", h.RemoveCartItem)

	// orders
	http.HandleFunc("POST /orders", h.CreateOrder)

	// items
	http.HandleFunc("GET /items", h.GetItems)
	http.HandleFunc("GET /items/{item_id}", h.GetItemByID)

	// users
	http.HandleFunc("POST /signup", h.CreateUser)
	http.HandleFunc("POST /login", h.LoginUser)
	http.HandleFunc("GET /token", h.IssueJWT)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func mustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		log.Fatalf("%s is not set", key)
	}
	return value
}
