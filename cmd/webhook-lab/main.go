// webhook-lab is the webhook receiver from the week 12 lecture 2 homework.
//
//	go run ./cmd/webhook-lab                      # dedup in memory (sync.Map)
//	$env:DEDUP="redis"; go run ./cmd/webhook-lab  # dedup in Redis (SET NX EX)
//
// Environment (all optional):
//
//	ADDR            listen address              (default :8090)
//	WEBHOOK_SECRET  shared HMAC secret          (default: a dev-only value)
//	DATA_DIR        where the .jsonl files go   (default webhook-lab-data)
//	DEDUP           memory | redis              (default memory)
//	REDIS_ADDR      Redis address               (default localhost:6379)
//	FAIL_HANDLER    fraud | email | analytics: that handler always fails (DLQ demo)
//	REPLAY_TOKEN    if set, POST /replay needs "Authorization: Bearer <token>"
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"checkout-api/internal/webhooklab"
	"checkout-api/internal/webhooklab/redisdedup"
)

// devSecret is only for running the lab locally; set WEBHOOK_SECRET for
// anything else. The sender uses the same default, so the demo just works.
const devSecret = "webhook-lab-dev-secret"

func main() {
	addr := env("ADDR", ":8090")
	secret := env("WEBHOOK_SECRET", devSecret)
	dataDir := env("DATA_DIR", "webhook-lab-data")
	failHandler := os.Getenv("FAIL_HANDLER")
	replayToken := os.Getenv("REPLAY_TOKEN")

	if secret == devSecret {
		log.Printf("WEBHOOK_SECRET not set: using the dev-only default")
	}

	eventLog, err := webhooklab.OpenJSONL(filepath.Join(dataDir, "events.jsonl"))
	if err != nil {
		log.Fatal(err)
	}
	defer eventLog.Close()

	deadLetters, err := webhooklab.OpenJSONL(filepath.Join(dataDir, "dead_letter.jsonl"))
	if err != nil {
		log.Fatal(err)
	}
	defer deadLetters.Close()

	// Task 3 vs task 5: same interface, different storage.
	var dedup webhooklab.Deduper = &webhooklab.MemoryDeduper{}
	if env("DEDUP", "memory") == "redis" {
		rd := redisdedup.New(env("REDIS_ADDR", "localhost:6379"), 24*time.Hour)
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := rd.Ping(pingCtx)
		cancel()
		if err != nil {
			log.Fatalf("%v (is it running? docker compose up -d redis)", err)
		}
		defer rd.Close()
		dedup = rd
		log.Printf("dedup: Redis SET NX EX (survives restarts, shared by instances)")
	} else {
		log.Printf("dedup: in-memory sync.Map (lost on restart)")
	}

	if failHandler != "" {
		log.Printf("FAIL_HANDLER=%s: that handler always fails -> dead_letter.jsonl", failHandler)
	}

	// Task 7 + 8: three mock services, 3 tries each, then the DLQ file.
	dispatcher := webhooklab.NewDispatcher(
		webhooklab.MockHandlers(failHandler, 300*time.Millisecond),
		deadLetters,
		webhooklab.DispatcherConfig{MaxAttempts: 3, RetryDelay: 200 * time.Millisecond},
	)

	receiver := &webhooklab.Receiver{
		Secret:      secret,
		Log:         eventLog,
		Dedup:       dedup,
		Dispatch:    dispatcher.Dispatch,
		ReplayToken: replayToken,
	}
	if replayToken == "" {
		log.Printf("REPLAY_TOKEN not set: /replay is open (fine for a local lab only)")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           receiver.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		log.Printf("webhook-lab listening on %s (POST /webhook, POST /replay), files in %s/", addr, dataDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	// Graceful shutdown: stop taking requests, then let the handlers finish
	// the events already in their channels.
	log.Printf("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	dispatcher.Close()
	log.Printf("bye")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
