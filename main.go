package main

import (
	"checkout-api/handlers"
	"checkout-api/middlewares"
	"checkout-api/store"

	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			"warning: .env file not found; using system environment variables",
		)
	}

	databaseURL := mustGetEnv("DATABASE_URL")
	jwtSecret := mustGetEnv("JWT_SECRET")
	frontendURL := mustGetEnv("FRONTEND_URL")

	jwtSecretBytes := []byte(jwtSecret)

	ctx := context.Background()

	// Create PostgreSQL connection pool
	pool, err := pgxpool.New(
		ctx,
		databaseURL,
	)
	if err != nil {
		log.Fatalf(
			"failed to create database pool: %v",
			err,
		)
	}

	defer pool.Close()

	// Check that database is reachable
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf(
			"failed to ping database: %v",
			err,
		)
	}

	postgresStore :=
		store.NewPostgresStore(pool)

	h := handlers.NewHandler(
		postgresStore,
		jwtSecretBytes,
	)

	// --------------------
	// CART
	// --------------------

	http.Handle(
		"GET /user/cart",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.GetUserCart,
			),
			jwtSecretBytes,
		),
	)

	http.Handle(
		"PATCH /user/cart/items/{item_id}",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.UpsertCartItem,
			),
			jwtSecretBytes,
		),
	)

	http.Handle(
		"DELETE /user/cart/items/{item_id}",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.RemoveCartItem,
			),
			jwtSecretBytes,
		),
	)

	// --------------------
	// ORDERS
	// --------------------

	http.Handle(
		"POST /orders",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.CreateOrder,
			),
			jwtSecretBytes,
		),
	)

	http.Handle(
		"GET /user/orders",
		middlewares.AuthMiddleware(
			http.HandlerFunc(
				h.GetUserOrders,
			),
			jwtSecretBytes,
		),
	)

	// --------------------
	// ITEMS
	// --------------------

	http.HandleFunc(
		"GET /items",
		h.GetItems,
	)

	http.HandleFunc(
		"GET /items/{item_id}",
		h.GetItemByID,
	)

	// --------------------
	// AUTH
	// --------------------

	http.HandleFunc(
		"POST /signup",
		h.CreateUser,
	)

	http.HandleFunc(
		"POST /login",
		h.LoginUser,
	)

	http.HandleFunc(
		"GET /token",
		h.IssueJWT,
	)

	// --------------------
	// SERVER
	// --------------------

	fmt.Println(
		"Server starting on :8080",
	)

	log.Fatal(
		http.ListenAndServe(
			":8080",
			middlewares.CORS(
				http.DefaultServeMux,
				frontendURL,
			),
		),
	)
}

func mustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)

	if !ok || value == "" {
		log.Fatalf(
			"%s is not set",
			key,
		)
	}

	return value
}
