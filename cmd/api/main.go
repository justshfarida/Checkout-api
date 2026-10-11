package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"checkout-api/internal/handlers"
	"checkout-api/internal/middlewares"
	"checkout-api/internal/repositories/postgres"
	"checkout-api/internal/services"
	"checkout-api/internal/telemetry"

	_ "checkout-api/docs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	// DATABASE
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()
	// Fail fast if the database is unreachable instead of hanging at startup.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// REPOSITORY
	repo := postgres.NewPostgresRepository(pool)

	// SERVICES
	authService := services.NewAuthService(repo)
	itemService := services.NewItemService(repo)

	cartService := services.NewCartService(
		repo,
		repo,
		repo,
	)

	tokenService := services.NewTokenService(
		repo,
		jwtSecretBytes,
	)

	paymentProcessor := services.MockPaymentProcessor{}

	xsollaService := services.NewXsollaService(
		xsollaProjectID,
		xsollaMerchantID,
		xsollaAPIKey,
		xsollaWebhookSecret,
		// After payment, send the user back to this environment's frontend.
		strings.TrimRight(frontendURL, "/")+"/payment/success",
	)

	orderService := services.NewOrderService(
		repo,
		repo,
		repo,
		paymentProcessor,
		xsollaService,
		repo,
	)

	// HANDLERS AND ROUTES
	h := handlers.NewHandler(
		authService,
		itemService,
		cartService,
		orderService,
		tokenService,
		xsollaService,
		jwtSecretBytes,
	)

	mux := h.Routes()
	mux.HandleFunc("GET /telemetry-test-error", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Intentional telemetry test error", http.StatusInternalServerError)
	})

	// TELEMETRY
	shutdownTelemetry, err := telemetry.Init(ctx)
	if err != nil {
		log.Fatalf("failed to initialize telemetry: %v", err)
	}
	//every request gets a span, (method, route,  status)
	instrumentedHandler := otelhttp.NewHandler(
		middlewares.CORS(mux, frontendURL),
		"http.server",
		otelhttp.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/health"
		}),
	)

	// SERVER
	server := &http.Server{
		Addr:              ":8080",
		Handler:           instrumentedHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second, // max time to read the whole request
		WriteTimeout:      15 * time.Second, // max time to write the response
		IdleTimeout:       60 * time.Second, // keep-alive connections
	}

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		log.Println("Server starting on :8080")
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-signalCtx.Done():
		log.Println("Stopping server...")

	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}

	// Finish active requests before flushing telemetry.
	serverShutdownCtx, cancelServer := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	if err := server.Shutdown(serverShutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
		// Force connections closed if graceful shutdown times out.
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("server close error: %v", closeErr)
		}
	}
	cancelServer()

	// Flush pending traces and metrics.
	telemetryShutdownCtx, cancelTelemetry := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	if err := shutdownTelemetry(telemetryShutdownCtx); err != nil {
		log.Printf("telemetry shutdown error: %v", err)
	}
	cancelTelemetry()
}

func mustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)

	if !ok || value == "" {
		log.Fatalf("%s is not set", key)
	}

	return value
}
