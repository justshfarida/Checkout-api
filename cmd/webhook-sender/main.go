// webhook-sender plays the webhook provider: it sends signed events to
// webhook-lab and retries with exponential backoff (task 4).
//
//	go run ./cmd/webhook-sender                     # one event
//	go run ./cmd/webhook-sender -count 3            # three different events
//	go run ./cmd/webhook-sender -id evt_1 -dup      # the same event twice
//	go run ./cmd/webhook-sender -bad-signature      # wrongly signed -> 401
//
// Uses WEBHOOK_SECRET, or the same dev-only default as webhook-lab.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"checkout-api/internal/webhooklab"
)

const devSecret = "webhook-lab-dev-secret" // must match cmd/webhook-lab

func main() {
	url := flag.String("url", "http://localhost:8090/webhook", "receiver URL")
	eventType := flag.String("event", "payment.completed", "event type")
	id := flag.String("id", "", "event_id (default: a new one per event)")
	count := flag.Int("count", 1, "how many events to send")
	dup := flag.Bool("dup", false, "send every event twice (idempotency demo)")
	badSig := flag.Bool("bad-signature", false, "sign with a wrong secret (expect 401)")
	attempts := flag.Int("attempts", 5, "max delivery attempts")
	baseDelay := flag.Duration("base-delay", 500*time.Millisecond, "first retry wait, doubled each time")
	flag.Parse()

	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = devSecret
	}

	sender := &webhooklab.Sender{URL: *url, Secret: secret, MaxAttempts: *attempts, BaseDelay: *baseDelay}
	ctx := context.Background()

	for i := 1; i <= *count; i++ {
		eventID := *id
		if eventID == "" {
			eventID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
		} else if *count > 1 {
			eventID = fmt.Sprintf("%s_%d", *id, i)
		}

		data, _ := json.Marshal(map[string]any{
			"order_id": fmt.Sprintf("order_%d", 400+i),
			"amount":   99.99,
			"currency": "USD",
			"user_id":  "usr_789",
		})
		e := webhooklab.Event{
			Event:     *eventType,
			EventID:   eventID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      data,
		}

		times := 1
		if *dup {
			times = 2
		}
		for n := 0; n < times; n++ {
			send(ctx, sender, e, *badSig)
		}
	}
}

func send(ctx context.Context, s *webhooklab.Sender, e webhooklab.Event, badSig bool) {
	var (
		tries int
		err   error
	)
	if badSig {
		body, _ := json.Marshal(e)
		tries, err = s.SendRaw(ctx, body, webhooklab.Sign("wrong-secret", body))
	} else {
		tries, err = s.Send(ctx, e)
	}
	if err != nil {
		log.Printf("%s: FAILED after %d attempt(s): %v", e.EventID, tries, err)
		return
	}
	log.Printf("%s: delivered (attempt %d)", e.EventID, tries)
}
