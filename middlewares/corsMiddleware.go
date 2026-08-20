package middlewares

import "net/http"

func CORS(
	next http.Handler,
	frontendURL string,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Access-Control-Allow-Origin",
			frontendURL,
		)

		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PATCH, DELETE, OPTIONS",
		)

		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization, Idempotency-Key, Refresh-Token",
		)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
