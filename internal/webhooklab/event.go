// Package webhooklab is the week 12 (lecture 2) webhooks homework: a small
// webhook receiver and sender that show the patterns from the lecture in
// their simple, in-process form. It is independent of the Xsolla webhook
// in internal/handlers, which uses Xsolla's own signature format.
//
//	sender --POST /webhook--> receiver: verify HMAC -> log to file -> dedup
//	                                     -> 200 OK -> fan-out (channels)
//	                                        fraud / email / analytics
//	                                        failed 3x -> dead_letter.jsonl
package webhooklab

import "encoding/json"

// Event is the webhook payload (lecture slide 23).
type Event struct {
	Event     string          `json:"event"`     // what happened, e.g. "payment.completed"
	EventID   string          `json:"event_id"`  // unique id: the idempotency key
	Timestamp string          `json:"timestamp"` // when it happened at the provider
	Data      json.RawMessage `json:"data,omitempty"`
}
