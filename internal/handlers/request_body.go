package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
)

// maxBodyBytes caps every request body the API reads. Our largest real body
// (login/signup/cart) is a few hundred bytes; 1 MB leaves plenty of room
// while stopping a client from making us buffer gigabytes.
const maxBodyBytes = 1 << 20

// decodeJSON reads a JSON body of at most 1 MB into dst and rejects fields
// the struct doesn't know. On failure it writes 413 (too large) or 400 (bad
// JSON) and returns false, so callers just do: if !decodeJSON(...) { return }.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if isBodyTooLarge(err) {
			writePayloadTooLarge(w)
			return false
		}
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return false
	}
	return true
}

func isBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

func writePayloadTooLarge(w http.ResponseWriter) {
	writeError(
		w,
		http.StatusRequestEntityTooLarge,
		"PAYLOAD_TOO_LARGE",
		"Request body must be at most 1 MB",
	)
}
