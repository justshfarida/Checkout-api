package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"checkout-api/internal/handlers"
	"checkout-api/internal/middlewares"
	"checkout-api/internal/repositories/postgres"
	"checkout-api/internal/services"

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

	// --------------------
	// DATABASE
	// --------------------

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

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf(
			"failed to ping database: %v",
			err,
		)
	}

	// --------------------
	// REPOSITORY
	// --------------------

	repo :=
		postgres.NewPostgresRepository(
			pool,
		)

	// --------------------
	// SERVICES
	// --------------------

	authService :=
		services.NewAuthService(
			repo,
		)

	itemService :=
		services.NewItemService(
			repo,
		)

	cartService :=
		services.NewCartService(
			repo,
			repo,
		)

	orderService :=
		services.NewOrderService(
			repo,
			repo,
		)

	// --------------------
	// HANDLER
	// --------------------

	h :=
		handlers.NewHandler(
			authService,
			itemService,
			cartService,
			orderService,
			jwtSecretBytes,
		)

	// --------------------
	// ROUTES
	// --------------------

	mux := h.Routes()

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
				mux,
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
