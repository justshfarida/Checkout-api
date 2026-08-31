package main

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/controllers/handlers"
	middleware2 "checkout-api/internal/controllers/middleware"
	"checkout-api/internal/repository"
	"checkout-api/internal/service"
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
		log.Println("no .env file found, relying on real environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	if dbURL == "" || jwtSecret == "" {
		log.Fatal("DATABASE_URL and JWT_SECRET must be set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to ping db: %v", err)
	}
	fmt.Println("successfully connected to db")

	authManager := auth.NewManager(jwtSecret)

	// Items

	itemsRepo := repository.NewItemsRepo(pool)
	itemsService := service.NewItemsService(itemsRepo)
	itemsHandler := handlers.NewItemsHandler(itemsService)

	paymentService := service.NewPaymentService()

	// Cart

	cartRepo := repository.NewCartRepo(pool)
	cartService := service.NewCartService(cartRepo, itemsRepo)
	cartHandler := handlers.NewCartHandler(cartService)

	// Orders

	idempotencyRepo := repository.NewIdempotencyRepo(pool)

	ordersRepo := repository.NewOrdersRepo(pool)
	ordersService := service.NewOrdersService(ordersRepo, cartRepo, itemsRepo, idempotencyRepo, paymentService)
	ordersHandler := handlers.NewOrdersHandler(ordersService)

	// Refresh Token

	refreshTokenRepo := repository.NewRefreshTokenRepo(pool)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepo, authManager)
	refreshTokenHandler := handlers.NewRefreshTokenHandler(refreshTokenService)

	// Users

	userRepo := repository.NewUserRepo(pool)
	userService := service.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService, refreshTokenService, authManager)

	requireAuth := middleware2.JWTAuth(authManager)

	// Routes ----------

	// Cart

	http.Handle("/user/cart", requireAuth(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				cartHandler.GetUserCart(w, r)
			case http.MethodPost:
				cartHandler.CreateUserCartAndAddItems(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})))

	http.Handle("PATCH /user/cart/items/{id}", requireAuth(http.HandlerFunc(cartHandler.UpdateItemQuantity)))
	http.Handle("DELETE /user/cart/items/{id}", requireAuth(http.HandlerFunc(cartHandler.DeleteItem)))

	// Items

	http.HandleFunc("GET /items", itemsHandler.GetItems)
	http.HandleFunc("GET /items/{id}", itemsHandler.GetItem)

	// Orders

	http.Handle("POST /user/orders", requireAuth(http.HandlerFunc(ordersHandler.CreateOrder)))
	http.Handle("GET /user/orders", requireAuth(http.HandlerFunc(ordersHandler.GetOrdersByUserID)))

	// Users

	http.HandleFunc("POST /signup", userHandler.SignUp)
	http.HandleFunc("POST /login", userHandler.Login)

	// Refresh token

	http.HandleFunc("POST /token", refreshTokenHandler.RefreshToken)

	allowedOrigin := os.Getenv("LOCAL_SERVER")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}
	corsHandler := middleware2.CORS(allowedOrigin)(http.DefaultServeMux)

	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
