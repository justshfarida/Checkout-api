package handlers

import (
	"checkout-api/internal/httperror"
	"checkout-api/internal/service"
	"checkout-api/internal/validation"
	"errors"
	"net/http"
)

type errMapping struct {
	sentinel error
	status   int
	code     string
}

var errMappings = []errMapping{
	{service.ErrItemNotFound, http.StatusNotFound, "item_not_found"},
	{service.ErrQuantityLimitExceeded, http.StatusBadRequest, "quantity_limit_exceeded"},
	{service.ErrInsufficientStock, http.StatusConflict, "insufficient_stock"},
	{service.ErrCartItemNotFound, http.StatusNotFound, "cart_item_not_found"},
	{service.ErrCartEmpty, http.StatusBadRequest, "cart_empty"},
	{service.ErrInvalidQuantity, http.StatusBadRequest, "invalid_quantity"},
	{service.ErrIdempotencyConflict, http.StatusConflict, "idempotency_conflict"},
	{service.ErrUsernameTaken, http.StatusConflict, "username_taken"},
	{service.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
	{service.ErrRefreshTokenNotFound, http.StatusUnauthorized, "invalid_refresh_token"},
	{service.ErrRefreshTokenRevoked, http.StatusUnauthorized, "invalid_refresh_token"},
	{service.ErrRefreshTokenExpired, http.StatusUnauthorized, "invalid_refresh_token"},
}

func WriteServiceError(w http.ResponseWriter, err error) {
	for _, m := range errMappings {
		if errors.Is(err, m.sentinel) {
			httperror.Write(w, m.status, m.code, err.Error(), nil)
			return
		}
	}
	httperror.Write(w, http.StatusInternalServerError, "internal_error", "something went wrong", nil)
}

func WriteValidationError(w http.ResponseWriter, message string) {
	httperror.Write(w, http.StatusBadRequest, "validation_error", message, nil)
}

func WriteUnauthorized(w http.ResponseWriter, message string) {
	httperror.Write(w, http.StatusUnauthorized, "unauthorized", message, nil)
}

func WriteMethodNotAllowed(w http.ResponseWriter) {
	httperror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", nil)
}

func WriteShapeErrors(w http.ResponseWriter, result *validation.Result) {
	httperror.Write(w, http.StatusBadRequest, "validation_error", "request failed validation", result.Errors)
}
