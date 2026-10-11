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

	if err := validateJWTSecret(jwtSecret); err != nil {
		log.Fatal(err)
	}

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

	// TELEMETRY
	// Set up before anything creates tracers, meters or loggers
	// (e.g. the item service's custom counter).
	shutdownTelemetry, err := telemetry.Init(ctx)
	if err != nil {
		log.Fatalf("failed to initialize telemetry: %v", err)
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
	// Refresh cookie is Secure (HTTPS only) unless explicitly disabled for
	// local development over plain HTTP. Anything but "false" keeps it on.
	h.SetSecureCookies(os.Getenv("COOKIE_SECURE") != "false")

	mux := h.Routes()
	mux.HandleFunc("GET /telemetry-test-error", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Intentional telemetry test error", http.StatusInternalServerError)
	})
	// Demo endpoint for the panic & recover pattern: the panic is caught by
	// WithRecover, the client gets a 500, and the server keeps running.
	mux.HandleFunc("GET /telemetry-test-panic", func(w http.ResponseWriter, r *http.Request) {
		panic("intentional test panic")
	})

	//every request gets a span, (method, route,  status)
	// WithRecover sits inside otelhttp, so a recovered panic is still recorded
	// on the request's span (status 500) and its log carries the trace ID.
	instrumentedHandler := otelhttp.NewHandler(
		middlewares.WithRecover(middlewares.SecurityHeaders(middlewares.CORS(mux, frontendURL))),
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

	// GRACEFUL SHUTDOWN, in order:
	// 1. stop accepting new requests and finish in-flight ones
	// 2. flush pending telemetry
	// 3. close database connections

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

	// Close database connections last, after in-flight requests have
	// finished. (The deferred pool.Close above is a safety net; calling
	// Close twice is safe.)
	pool.Close()
	log.Println("Shutdown complete")
}

func mustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)

	if !ok || value == "" {
		log.Fatalf("%s is not set", key)
	}

	return value
}
