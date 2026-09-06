package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"checkout-api/internal/models"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestHandler_LoginUser(t *testing.T) {
	password := "correcthorse"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	internalErr := errors.New("db connection reset by peer")

	tests := []struct {
		name       string
		body       string
		svc        *fakeAuthService
		wantStatus int
	}{
		{
			name:       "ok: valid credentials",
			body:       `{"email":"ada@example.com","password":"correcthorse"}`,
			svc:        &fakeAuthService{user: models.User{ID: 1, Email: "ada@example.com", Hash: hash}},
			wantStatus: http.StatusOK,
		},
		{name: "bad request: malformed json", body: `{not-json`, svc: &fakeAuthService{}, wantStatus: http.StatusBadRequest},
		{name: "validation: invalid email", body: `{"email":"bad-email","password":"correcthorse"}`, svc: &fakeAuthService{}, wantStatus: http.StatusUnprocessableEntity},
		{name: "unauthorized: user not found", body: `{"email":"ada@example.com","password":"correcthorse"}`, svc: &fakeAuthService{findErr: pgx.ErrNoRows}, wantStatus: http.StatusUnauthorized},
		{name: "unauthorized: wrong password", body: `{"email":"ada@example.com","password":"wrongpass"}`, svc: &fakeAuthService{user: models.User{ID: 1, Email: "ada@example.com", Hash: hash}}, wantStatus: http.StatusUnauthorized},
		{name: "unknown service error maps to 500", body: `{"email":"ada@example.com","password":"correcthorse"}`, svc: &fakeAuthService{findErr: internalErr}, wantStatus: http.StatusInternalServerError},
		{name: "save refresh token error maps to 500", body: `{"email":"ada@example.com","password":"correcthorse"}`, svc: &fakeAuthService{user: models.User{ID: 1, Email: "ada@example.com", Hash: hash}, saveRefreshErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(tt.svc, nil, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				if !tt.svc.saveRefreshCalled {
					t.Fatal("expected SaveRefreshToken to be called")
				}
				found := false
				for _, cookie := range w.Result().Cookies() {
					if cookie.Name == "refresh_token" {
						found = true
						if !cookie.HttpOnly {
							t.Fatal("refresh_token cookie must be HttpOnly")
						}
					}
				}
				if !found {
					t.Fatal("refresh_token cookie was not set")
				}
				if !strings.Contains(w.Body.String(), `"jwt"`) {
					t.Fatalf("response does not contain jwt: %s", w.Body.String())
				}
			}
		})
	}
}

func TestHandler_CreateUser(t *testing.T) {
	internalErr := errors.New("insert failed")
	tests := []struct {
		name       string
		body       string
		svc        *fakeAuthService
		wantStatus int
	}{
		{name: "ok: user created", body: `{"email":"ada@example.com","password":"correcthorse"}`, svc: &fakeAuthService{}, wantStatus: http.StatusCreated},
		{name: "bad request: malformed json", body: `{not-json`, svc: &fakeAuthService{}, wantStatus: http.StatusBadRequest},
		{name: "validation: short password", body: `{"email":"ada@example.com","password":"short"}`, svc: &fakeAuthService{}, wantStatus: http.StatusUnprocessableEntity},
		{name: "unknown service error maps to 500", body: `{"email":"ada@example.com","password":"correcthorse"}`, svc: &fakeAuthService{saveUserErr: internalErr}, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(tt.svc, nil, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusCreated {
				if !tt.svc.saveUserCalled {
					t.Fatal("expected SaveUser to be called")
				}
				if err := bcrypt.CompareHashAndPassword(tt.svc.savedHash, []byte("correcthorse")); err != nil {
					t.Fatalf("saved password is not the expected bcrypt hash: %v", err)
				}
			}
		})
	}
}

func TestHandler_IssueJWT(t *testing.T) {
	tests := []struct {
		name       string
		cookie     *http.Cookie
		svc        *fakeAuthService
		wantStatus int
	}{
		{name: "ok: valid refresh token", cookie: &http.Cookie{Name: "refresh_token", Value: "old-refresh-token"}, svc: &fakeAuthService{rotateUserID: 1}, wantStatus: http.StatusOK},
		{name: "unauthorized: missing refresh token", svc: &fakeAuthService{}, wantStatus: http.StatusUnauthorized},
		{name: "unauthorized: empty refresh token", cookie: &http.Cookie{Name: "refresh_token", Value: ""}, svc: &fakeAuthService{}, wantStatus: http.StatusUnauthorized},
		{name: "unauthorized: invalid refresh token", cookie: &http.Cookie{Name: "refresh_token", Value: "invalid"}, svc: &fakeAuthService{rotateErr: errors.New("expired")}, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(tt.svc, nil, nil, nil)
			req := httptest.NewRequest(http.MethodGet, "/token", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				if !tt.svc.rotateCalled {
					t.Fatal("expected RotateRefreshToken to be called")
				}
				if !strings.Contains(w.Body.String(), `"jwt"`) {
					t.Fatalf("response does not contain jwt: %s", w.Body.String())
				}
			}
		})
	}
}
