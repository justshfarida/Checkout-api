package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"checkout-api/internal/services"
	"checkout-api/internal/validation"

	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	Code    string        `json:"error"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	details ...ErrorDetail,
) {
	writeJSON(
		w,
		status,
		ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	)
}

func handleError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {

	case errors.Is(
		err,
		services.ErrItemNotFound,
	):
		writeError(
			w,
			http.StatusNotFound,
			"ITEM_NOT_FOUND",
			"Item was not found",
		)

	case errors.Is(
		err,
		validation.ErrInsufficientStock,
	):
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"INSUFFICIENT_STOCK",
			"Requested quantity exceeds available stock",
		)

	case errors.Is(
		err,
		services.ErrInvalidCursor,
	):
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_CURSOR",
			"Cursor is invalid",
		)

	default:
		// Keep the real internal error in server logs. Using the request
		// context attaches trace_id/span_id, so the log links to its trace.
		slog.ErrorContext(
			r.Context(),
			"Request failed",
			"error", err.Error(),
			"method", r.Method,
			"path", r.URL.Path,
		)

		// Never expose database/internal implementation details.
		writeError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Something went wrong on our side",
		)
	}
}
func validationDetails(
	err error,
) []ErrorDetail {
	validationErrors, ok :=
		err.(validator.ValidationErrors)

	if !ok {
		return nil
	}

	details :=
		make(
			[]ErrorDetail,
			0,
			len(validationErrors),
		)

	for _, fieldError := range validationErrors {

		details = append(
			details,
			ErrorDetail{
				Field:   fieldError.Field(),
				Message: validationMessage(fieldError),
			},
		)
	}

	return details
}
func validationMessage(
	err validator.FieldError,
) string {
	switch err.Tag() {
	case "required":
		return "This field is required"

	case "email":
		return "Must be a valid email address"

	case "gt":
		return "Must be greater than " + err.Param()

	case "min":
		return "Must be at least " + err.Param()

	case "max":
		return "Must be at most " + err.Param()

	default:
		return "Invalid value"
	}
}
