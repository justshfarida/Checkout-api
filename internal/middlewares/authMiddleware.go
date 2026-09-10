package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "userID"

type errorResponse struct {
	Code    string `json:"error"`
	Message string `json:"message"`
}

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(
		errorResponse{
			Code:    code,
			Message: message,
		},
	)
}

func AuthMiddleware(
	next http.Handler,
	jwtSecret []byte,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			authorizationHeader :=
				r.Header.Get("Authorization")

			if authorizationHeader == "" {
				writeError(
					w,
					http.StatusUnauthorized,
					"UNAUTHORIZED",
					"Authentication is required",
				)
				return
			}

			const scheme = "bearer "

			if len(authorizationHeader) < len(scheme) {
				writeError(
					w,
					http.StatusUnauthorized,
					"UNAUTHORIZED",
					"Authentication is required",
				)
				return
			}

			userScheme :=
				authorizationHeader[:len(scheme)]

			if !strings.EqualFold(
				scheme,
				userScheme,
			) {
				writeError(
					w,
					http.StatusUnauthorized,
					"INVALID_AUTH_SCHEME",
					"Authorization header must use Bearer scheme",
				)
				return
			}

			tokenString :=
				strings.TrimSpace(
					authorizationHeader[len(scheme):],
				)

			if tokenString == "" {
				writeError(
					w,
					http.StatusUnauthorized,
					"INVALID_TOKEN",
					"Access token is missing",
				)
				return
			}

			var claims jwt.RegisteredClaims

			token, err :=
				jwt.ParseWithClaims(
					tokenString,
					&claims,
					func(token *jwt.Token) (any, error) {
						return jwtSecret, nil
					},
					jwt.WithValidMethods(
						[]string{"HS256"},
					),
				)

			if err != nil || !token.Valid {
				fmt.Printf(
					"failed to parse JWT: %v\n",
					err,
				)

				writeError(
					w,
					http.StatusUnauthorized,
					"INVALID_TOKEN",
					"Access token is invalid or expired",
				)
				return
			}

			userID, err :=
				strconv.Atoi(
					claims.Subject,
				)

			if err != nil || userID <= 0 {
				fmt.Printf(
					"failed to parse user ID: %v\n",
					err,
				)

				writeError(
					w,
					http.StatusUnauthorized,
					"INVALID_TOKEN",
					"Access token contains invalid user information",
				)
				return
			}

			ctx :=
				context.WithValue(
					r.Context(),
					userIDContextKey,
					userID,
				)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}

func UserIDFromContext(
	ctx context.Context,
) (int, bool) {
	userID, ok :=
		ctx.Value(
			userIDContextKey,
		).(int)

	return userID, ok
}
