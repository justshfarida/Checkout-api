package middlewares

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// WithRecover turns a panic in any handler into a 500 response instead of
// crashing the request goroutine. The panic and its stack trace are logged
// with the request context (so the log carries trace_id) and recorded on the
// current span, and the server keeps serving other requests.
func WithRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			// http.ErrAbortHandler is net/http's deliberate way to abort a
			// response; let the server handle it as usual.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			panicErr := fmt.Errorf("panic: %v", rec)

			span := trace.SpanFromContext(r.Context())
			span.RecordError(panicErr)
			span.SetStatus(codes.Error, "panic recovered")

			slog.ErrorContext(r.Context(), "Panic recovered",
				"panic", fmt.Sprint(rec),
				"method", r.Method,
				"path", r.URL.Path,
				"stack", string(debug.Stack()),
			)

			writeError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Something went wrong on our side",
			)
		}()

		next.ServeHTTP(w, r)
	})
}
