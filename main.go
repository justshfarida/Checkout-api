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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set (check your .env file)")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set (check your .env file)")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		panic(err)
	}

	if err := conn.Ping(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected to db")

	postgresStore := store.NewPostgresStore(conn)
	handler := handlers.NewHandler(postgresStore)

	mux := http.NewServeMux()

	// Items
	http.HandleFunc("GET /items", handler.GetItems)
	http.HandleFunc("GET /items/{item_id}", handler.GetItem)

	// Users
	http.HandleFunc("POST /signup", handler.SignUp)
	http.HandleFunc("POST /login", handler.Login)
	http.HandleFunc("GET /token", handler.Token)

	// Cart
	http.HandleFunc("POST /user/cart", handler.CreateUserCartAndAddItems)
	// http.Handle("GET /user/cart", handler.JWTMiddleware(handler.GetUserCart))
	http.HandleFunc("PATCH /user/cart/items/{item_id}", handler.UpsertCartItem)
	// http.HandleFunc("DELETE /user/cart/items/{item_id}", handler.JWTMiddleware(handler.RemoveCartItem))

	// Orders
	// http.HandleFunc("POST /orders", handler.JWTMiddleware(handler.CreateOrder))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
