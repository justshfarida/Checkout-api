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
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// s := store.NewInMemStore()
	postgresStore := store.NewPostgresStore(conn)
	h := handlers.NewHandler(postgresStore, []byte(jwtSecret))

	// cart
	http.Handle(
		"GET /user/cart",
		h.AuthMiddleware(http.HandlerFunc(h.GetUserCart)),
	)

	http.Handle(
		"PATCH /user/cart/items/{item_id}",
		h.AuthMiddleware(http.HandlerFunc(h.UpsertCartItem)),
	)

	http.Handle(
		"DELETE /user/cart/items/{item_id}",
		h.AuthMiddleware(http.HandlerFunc(h.RemoveCartItem)),
	)

	http.Handle(
		"POST /orders",
		h.AuthMiddleware(http.HandlerFunc(h.CreateOrder)),
	)
	http.HandleFunc("GET /items", h.GetItems)
	http.HandleFunc("GET /items/{item_id}", h.GetItemByID)
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
