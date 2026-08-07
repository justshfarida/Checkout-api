package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func GetUserID(ctx context.Context) int {
	return ctx.Value(userIDKey).(int)
}

func Auth(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or malformed Authorization header", http.StatusUnauthorized)
				return
			}
			rawToken := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.ParseWithClaims(
				rawToken,
				&jwt.RegisteredClaims{},
				func(token *jwt.Token) (any, error) {
					return []byte(jwtSecret), nil
				},
			)

			if err != nil || !token.Valid {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			claims := token.Claims.(*jwt.RegisteredClaims)

			userID, err := strconv.Atoi(claims.Subject)
			if err != nil {
				http.Error(w, "invalid subject", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userID, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
