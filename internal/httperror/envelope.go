package httperror

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Error Detail `json:"error"`
}

type Detail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func Write(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{Error: Detail{Code: code, Message: message, Fields: fields}})
}
