package main

import (
	"checkout-api/auth"
	"checkout-api/handlers"
	"checkout-api/middleware"
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
	/*
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("CATCH-ALL: %s %s\n", r.Method, r.URL.Path)
			http.NotFound(w, r)
		})
	*/

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	if dbURL == "" || jwtSecret == "" {
		log.Fatal("DATABASE_URL and JWT_SECRET must be set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		panic(err)
	}
	err = conn.Ping(ctx)
	if err != nil {
		panic(err)
	}

	// defer conn.Close()

	fmt.Println("successfully connected to db")

	postgresStore := store.NewPostgresStore(conn)
	authManager := auth.NewManager(jwtSecret)
	handler := handlers.NewHandler(postgresStore, authManager)
	requireAuth := middleware.JWTAuth(authManager)

	// Cart
	http.Handle("/user/cart", requireAuth(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.GetUserCart(w, r)
			case http.MethodPost:
				handler.CreateUserCartAndAddItems(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})))

	http.Handle("PATCH /user/cart/items/{id}", requireAuth(http.HandlerFunc(handler.UpdateItemQuantity)))
	http.Handle("DELETE /user/cart/items/{id}", requireAuth(http.HandlerFunc(handler.DeleteItem)))

	// Items
	http.HandleFunc("GET /items", handler.GetItems)
	http.HandleFunc("GET /items/", handler.GetItem)

	// Orders
	http.Handle("POST /orders", requireAuth(http.HandlerFunc(handler.CreateOrder)))
	http.HandleFunc("GET /users/{user_id}/orders", handler.GetOrdersByUserID)

	// Users
	http.HandleFunc("POST /signup", handler.SignUp)
	http.HandleFunc("POST /login", handler.Login)
	// http.HandleFunc("GET /token", handler.RefreshToken)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
