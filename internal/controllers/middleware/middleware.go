package middleware

import (
	"checkout-api/internal/auth"
	"checkout-api/internal/httperror"
	"net/http"
	"strings"
)

func JWTAuth(manager *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				httperror.Write(w, http.StatusUnauthorized, "unauthorized", "missing or malformed Authorization header", nil)
				return
			}

			claims, err := manager.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				httperror.Write(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token", nil)
				return
			}

			ctx := auth.ContextWithUserID(r.Context(), claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
