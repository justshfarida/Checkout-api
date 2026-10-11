package middlewares

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds browser security headers to every response.
//
//   - Strict-Transport-Security: once a browser has seen us over HTTPS, it
//     refuses plain HTTP for a year (no downgrade / cookie sniffing).
//   - X-Content-Type-Options: nosniff - the browser trusts our Content-Type
//     instead of guessing (a JSON error can't be run as a script).
//   - X-Frame-Options: DENY and CSP frame-ancestors 'none' - nobody can embed
//     the API in an iframe (clickjacking).
//   - Content-Security-Policy: default-src 'none' - JSON needs no scripts,
//     styles or images, so the browser may load nothing.
//   - Referrer-Policy: no-referrer - URLs never leak to other sites.
//   - Cache-Control: no-store - responses with user data are never cached.
//
// The Swagger UI under /swagger/ is an HTML page that needs its own scripts
// and styles, so it skips the strict CSP and no-store.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")

		if !strings.HasPrefix(r.URL.Path, "/swagger/") {
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			h.Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}
