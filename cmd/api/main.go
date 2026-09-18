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

	_ "checkout-api/docs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// @title Checkout API
// @version 1.0
// @description API for items, cart, orders and authentication.
// @host localhost:8080
// @BasePath /

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			"warning: .env file not found; using system environment variables",
		)
	}

	databaseURL := mustGetEnv("DATABASE_URL")
	jwtSecret := mustGetEnv("JWT_SECRET")
	frontendURL := mustGetEnv("FRONTEND_URL")
	xsollaProjectID := mustGetEnv("XSOLLA_PROJECT_ID")
	xsollaMerchantID := mustGetEnv("XSOLLA_MERCHANT_ID")
	xsollaAPIKey := mustGetEnv("XSOLLA_API_KEY")
	xsollaWebhookSecret := mustGetEnv("XSOLLA_WEBHOOK_SECRET")

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
			repo,
		)
	tokenService := services.NewTokenService(
		repo,
		jwtSecretBytes,
	)
	paymentProcessor :=
		services.MockPaymentProcessor{}
	xsollaService := services.NewXsollaService(
		xsollaProjectID,
		xsollaMerchantID,
		xsollaAPIKey,
		xsollaWebhookSecret,
	)
	orderService :=
		services.NewOrderService(
			repo,
			repo,
			repo,
			paymentProcessor,
			xsollaService,
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
			tokenService,
			xsollaService,
			jwtSecretBytes,
			xsollaWebhookSecret,
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
