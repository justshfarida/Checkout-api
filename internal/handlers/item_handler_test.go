package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"checkout-api/internal/models"
	"checkout-api/internal/services"
)

func TestHandler_GetItemByID_MapsErrorsToStatus(t *testing.T) {
	internalErr := errors.New(
		"database connection failed: password=secret",
	)

	tests := []struct {
		name       string
		svc        *fakeItemService
		path       string
		wantStatus int
	}{
		{
			name: "ok: item found",
			svc: &fakeItemService{
				item: &models.Item{
					ID:   1,
					Name: "Keyboard",
				},
			},
			path:       "/items/1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "bad request: invalid item id",
			svc:        &fakeItemService{},
			path:       "/items/abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found: item does not exist",
			svc: &fakeItemService{
				getItemErr: services.ErrItemNotFound,
			},
			path:       "/items/1",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "unknown error: maps to 500",
			svc: &fakeItemService{
				getItemErr: internalErr,
			},
			path:       "/items/1",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(
				nil,
				tt.svc,
				nil,
				nil,
				nil,
			)

			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			w := httptest.NewRecorder()

			h.Routes().ServeHTTP(
				w,
				req,
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}

			if tt.wantStatus ==
				http.StatusInternalServerError &&
				strings.Contains(
					w.Body.String(),
					internalErr.Error(),
				) {

				t.Fatalf(
					"internal error leaked in response: %q",
					w.Body.String(),
				)
			}
		})
	}
}

func TestHandler_GetItems(t *testing.T) {
	internalErr := errors.New(
		"database unavailable",
	)

	tests := []struct {
		name       string
		path       string
		svc        *fakeItemService
		wantStatus int
	}{
		{
			name: "ok: list items",
			path: "/items?limit=10&offset=0",
			svc: &fakeItemService{
				offsetResult: services.ItemOffsetResult{
					Items: []*models.Item{
						{
							ID:   1,
							Name: "Keyboard",
						},
					},
					Limit:  10,
					Offset: 0,
					Total:  1,
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "bad request: limit is not an integer",
			path:       "/items?limit=abc",
			svc:        &fakeItemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request: negative offset",
			path:       "/items?offset=-1",
			svc:        &fakeItemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unknown service error maps to 500",
			path: "/items",
			svc: &fakeItemService{
				offsetErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(
				nil,
				tt.svc,
				nil,
				nil,
				nil,
			)

			w := httptest.NewRecorder()

			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			h.Routes().ServeHTTP(
				w,
				req,
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}
		})
	}
}

func TestHandler_GetItemsCursor(t *testing.T) {
	internalErr := errors.New(
		"database unavailable",
	)

	tests := []struct {
		name       string
		path       string
		svc        *fakeItemService
		wantStatus int
	}{
		{
			name: "ok: cursor page",
			path: "/items/cursor?limit=10",
			svc: &fakeItemService{
				cursorResult: services.ItemCursorResult{
					Items: []*models.Item{
						{
							ID: 1,
						},
					},
					HasMore: false,
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "bad request: invalid limit",
			path:       "/items/cursor?limit=abc",
			svc:        &fakeItemService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "bad request: invalid cursor from service",
			path: "/items/cursor?cursor=bad",
			svc: &fakeItemService{
				cursorErr: services.ErrInvalidCursor,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unknown service error maps to 500",
			path: "/items/cursor",
			svc: &fakeItemService{
				cursorErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(
				nil,
				tt.svc,
				nil,
				nil,
				nil,
			)

			w := httptest.NewRecorder()

			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			h.Routes().ServeHTTP(
				w,
				req,
			)

			if w.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					w.Code,
					tt.wantStatus,
					w.Body.String(),
				)
			}
		})
	}
}
