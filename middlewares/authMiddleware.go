package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "userID"

func AuthMiddleware(
	next http.Handler,
	jwtSecret []byte,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			http.Error(
				w,
				"Unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		const scheme = "bearer "

		if len(authorizationHeader) < len(scheme) {
			http.Error(
				w,
				"Unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		userScheme := authorizationHeader[:len(scheme)]

		if !strings.EqualFold(scheme, userScheme) {
			http.Error(
				w,
				"Unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		tokenString := authorizationHeader[len(scheme):]

		var claims jwt.RegisteredClaims

		token, err := jwt.ParseWithClaims(
			tokenString,
			&claims,
			func(token *jwt.Token) (any, error) {
				return jwtSecret, nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)

		if err != nil || !token.Valid {
			fmt.Printf(
				"failed to parse JWT: %v\n",
				err,
			)

			http.Error(
				w,
				"Unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		userID, err := strconv.Atoi(claims.Subject)

		if err != nil {
			fmt.Printf(
				"failed to parse user ID: %v\n",
				err,
			)

			http.Error(
				w,
				"Unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			userIDContextKey,
			userID,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

func UserIDFromContext(
	ctx context.Context,
) (int, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int)

	return userID, ok
}
