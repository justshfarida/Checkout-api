package webhooklab

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// maxBodyBytes caps a webhook body: nobody can fill our memory or log file
// with a huge request.
const maxBodyBytes = 1 << 20 // 1 MiB

// Receiver is the HTTP side: POST /webhook and POST /replay.
type Receiver struct {
	Secret      string     // shared HMAC secret (task 2)
	Log         *JSONLFile // event log (task 1)
	Dedup       Deduper    // memory (task 3) or Redis (task 5)
	Dispatch    func(Event) error
	ReplayToken string // if set, /replay needs "Authorization: Bearer <token>"
}

// Routes returns the receiver's endpoints. The method in the pattern makes
// Go answer 405 Method Not Allowed for anything but POST.
func (rc *Receiver) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook", rc.handleWebhook)
	mux.HandleFunc("POST /replay", rc.handleReplay)
	return mux
}

// handleWebhook: verify -> parse -> log -> dedup -> hand off -> 200.
// It answers fast and does the real work in the background (slide 28):
// the provider's timeout is on our HTTP response, not on our processing.
func (rc *Receiver) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "body too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read body"})
		return
	}

	// Task 2: reject anything not signed with our secret, BEFORE trusting
	// or even parsing the body. The HMAC is over the raw bytes, exactly as
	// they arrived.
	sig := r.Header.Get(SignatureHeader)
	if sig == "" {
		log.Printf("rejected: missing %s header", SignatureHeader)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing signature"})
		return
	}
	if !Verify(rc.Secret, body, sig) {
		log.Printf("rejected: invalid signature")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
		return
	}

	var e Event
	if err := json.Unmarshal(body, &e); err != nil || e.EventID == "" || e.Event == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body must be JSON with event and event_id"})
		return
	}

	// Task 1: every accepted event goes to the log file, with the time we
	// received it. Written before anything else happens, so /replay can
	// always re-process it. If the write fails we answer 500 and the
	// provider retries: better than accepting an event we did not record.
	entry := LogEntry{ReceivedAt: time.Now().UTC(), EventID: e.EventID, Event: e.Event, Payload: body}
	if err := rc.Log.Append(entry); err != nil {
		log.Printf("event log: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not record event"})
		return
	}

	// Tasks 3/5: the same event_id twice is processed once. A duplicate
	// still gets 200: an error would make the provider retry it forever.
	first, err := rc.Dedup.FirstSeen(r.Context(), e.EventID)
	if err != nil {
		// e.g. Redis down: we cannot know if it is a duplicate, so we do not
		// guess. 503 -> the provider retries later.
		log.Printf("dedup: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "try again later"})
		return
	}
	if !first {
		log.Printf("duplicate %s: already received, skipping", e.EventID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	// Task 7: hand off to the fan-out and answer right away.
	if err := rc.Dispatch(e); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "shutting down"})
		return
	}
	log.Printf("accepted %s (%s)", e.EventID, e.Event)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

// handleReplay re-processes events from the log file (task 6). It skips
// the dedup check on purpose: replaying means "run it again", e.g. after a
// downstream bug was fixed. Each event id is replayed once, even if the
// provider delivered it several times.
//
//	POST /replay                  every event in the log
//	POST /replay?event_id=evt_1   only that one
func (rc *Receiver) handleReplay(w http.ResponseWriter, r *http.Request) {
	if rc.ReplayToken != "" && !validBearer(r.Header.Get("Authorization"), rc.ReplayToken) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or wrong replay token"})
		return
	}

	entries, err := ReadEventLog(rc.Log.Path())
	if err != nil {
		log.Printf("replay: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot read event log"})
		return
	}

	only := r.URL.Query().Get("event_id")
	done := map[string]bool{}
	replayed := 0
	for _, entry := range entries {
		if done[entry.EventID] || (only != "" && entry.EventID != only) {
			continue
		}
		done[entry.EventID] = true

		var e Event
		if err := json.Unmarshal(entry.Payload, &e); err != nil {
			log.Printf("replay: skip %s: %v", entry.EventID, err)
			continue
		}
		if err := rc.Dispatch(e); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "shutting down", "replayed": replayed})
			return
		}
		replayed++
	}

	if only != "" && replayed == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event_id not in log"})
		return
	}
	log.Printf("replayed %d event(s) from %s", replayed, rc.Log.Path())
	writeJSON(w, http.StatusOK, map[string]int{"replayed": replayed})
}

// validBearer compares "Bearer <token>" in constant time.
func validBearer(header, token string) bool {
	got, ok := strings.CutPrefix(header, "Bearer ")
	return ok && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
