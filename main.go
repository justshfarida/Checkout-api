package main

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/handlers"
	"checkout-api/internal/middleware"
	"checkout-api/repository"
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

	fmt.Println("successfully connected to db")

	postgresStore := repository.NewPostgresStore(conn)
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

	allowedOrigin := os.Getenv("LOCAL_SERVER")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}
	corsHandler := middleware.CORS(allowedOrigin)(http.DefaultServeMux)

	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
