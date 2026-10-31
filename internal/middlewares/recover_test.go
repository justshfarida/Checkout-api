package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithRecover_PanicBecomes500AndServerKeepsServing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) {
		panic("something broke")
	})
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := WithRecover(mux)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panicking handler status = %d, want 500", w.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != "INTERNAL_ERROR" {
		t.Errorf("body = %q (err %v), want JSON with error INTERNAL_ERROR", w.Body.String(), err)
	}
	if got := w.Body.String(); strings.Contains(got, "something broke") {
		t.Errorf("response leaks the panic message: %s", got)
	}

	// The same handler chain must keep working after a panic.
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("request after panic status = %d, want 200", w2.Code)
	}
}

func TestWithRecover_PassesThroughNormalResponses(t *testing.T) {
	h := WithRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders", nil))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 unchanged", w.Code)
	}
}

func TestWithRecover_LetsErrAbortHandlerPropagate(t *testing.T) {
	h := WithRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler to be re-panicked", rec)
		}
	}()

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	t.Fatal("expected ErrAbortHandler to propagate")
}
