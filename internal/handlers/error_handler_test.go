package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"checkout-api/internal/services"
	"checkout-api/internal/validation"
)

func TestHandleError(t *testing.T) {
	internalErr :=
		errors.New("database connection failed: password=secret")

	tests := []struct {
		name string
		err  error

		wantStatus  int
		wantCode    string
		wantMessage string

		mustNotContain string
	}{
		{
			name: "item not found returns 404",

			err: services.ErrItemNotFound,

			wantStatus: http.StatusNotFound,

			wantCode: "ITEM_NOT_FOUND",

			wantMessage: "Item was not found",
		},
		{
			name: "invalid cursor returns 400",

			err: services.ErrInvalidCursor,

			wantStatus: http.StatusBadRequest,

			wantCode: "INVALID_CURSOR",

			wantMessage: "Cursor is invalid",
		},
		{
			name: "insufficient stock returns 422",

			err: validation.ErrInsufficientStock,

			wantStatus: http.StatusUnprocessableEntity,

			wantCode: "INSUFFICIENT_STOCK",

			wantMessage: "Requested quantity exceeds available stock",
		},
		{
			name: "unknown error returns 500 without leaking internal message",

			err: internalErr,

			wantStatus: http.StatusInternalServerError,

			wantCode: "INTERNAL_ERROR",

			wantMessage: "Something went wrong on our side",

			mustNotContain: internalErr.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				// ARRANGE
				recorder :=
					httptest.NewRecorder()

				// ACT
				handleError(
					recorder,
					httptest.NewRequest(http.MethodGet, "/items", nil),
					tt.err,
				)

				// ASSERT STATUS
				if recorder.Code !=
					tt.wantStatus {

					t.Errorf(
						"status = %d, want %d",
						recorder.Code,
						tt.wantStatus,
					)
				}

				body :=
					recorder.Body.String()

				// ASSERT ERROR CODE
				if !strings.Contains(
					body,
					tt.wantCode,
				) {
					t.Errorf(
						"response body = %q, want code %q",
						body,
						tt.wantCode,
					)
				}

				// ASSERT PUBLIC MESSAGE
				if !strings.Contains(
					body,
					tt.wantMessage,
				) {
					t.Errorf(
						"response body = %q, want message %q",
						body,
						tt.wantMessage,
					)
				}

				// ASSERT INTERNAL ERROR IS NOT LEAKED
				if tt.mustNotContain != "" &&
					strings.Contains(
						body,
						tt.mustNotContain,
					) {

					t.Errorf(
						"response leaked internal error: %q",
						tt.mustNotContain,
					)
				}
			},
		)
	}
}
