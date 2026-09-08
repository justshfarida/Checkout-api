package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checkout-api/internal/models"
	"checkout-api/internal/services"
)

func TestHandler_LoginUser(t *testing.T) {
	internalErr := errors.New(
		"db connection reset by peer",
	)

	tests := []struct {
		name       string
		body       string
		auth       *fakeAuthService
		token      *fakeTokenService
		wantStatus int
	}{
		{
			name: "ok: valid credentials",
			body: `{
				"email":"ada@example.com",
				"password":"correcthorse"
			}`,
			auth: &fakeAuthService{
				authenticateUser: models.User{
					ID:    1,
					Email: "ada@example.com",
				},
			},
			token: &fakeTokenService{
				accessToken:  "access-token",
				refreshToken: "refresh-token",
				refreshExpiresAt: time.Now().Add(
					30 * 24 * time.Hour,
				),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "bad request: malformed json",
			body:       `{not-json`,
			auth:       &fakeAuthService{},
			token:      &fakeTokenService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation: invalid email",
			body: `{
				"email":"bad-email",
				"password":"correcthorse"
			}`,
			auth:       &fakeAuthService{},
			token:      &fakeTokenService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "unauthorized: invalid credentials",
			body: `{
				"email":"ada@example.com",
				"password":"wrongpass"
			}`,
			auth: &fakeAuthService{
				authenticateErr: services.ErrInvalidCredentials,
			},
			token:      &fakeTokenService{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unknown auth service error maps to 500",
			body: `{
				"email":"ada@example.com",
				"password":"correcthorse"
			}`,
			auth: &fakeAuthService{
				authenticateErr: internalErr,
			},
			token:      &fakeTokenService{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "access token error maps to 500",
			body: `{
				"email":"ada@example.com",
				"password":"correcthorse"
			}`,
			auth: &fakeAuthService{
				authenticateUser: models.User{
					ID:    1,
					Email: "ada@example.com",
				},
			},
			token: &fakeTokenService{
				accessTokenErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "refresh token error maps to 500",
			body: `{
				"email":"ada@example.com",
				"password":"correcthorse"
			}`,
			auth: &fakeAuthService{
				authenticateUser: models.User{
					ID:    1,
					Email: "ada@example.com",
				},
			},
			token: &fakeTokenService{
				accessToken:     "access-token",
				refreshTokenErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				h := newTestHandler(
					tt.auth,
					nil,
					nil,
					nil,
					tt.token,
				)

				req := httptest.NewRequest(
					http.MethodPost,
					"/login",
					strings.NewReader(tt.body),
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

				if tt.wantStatus != http.StatusOK {
					return
				}

				if !tt.auth.authenticateCalled {
					t.Fatal(
						"expected Authenticate to be called",
					)
				}

				if !tt.token.createAccessCalled {
					t.Fatal(
						"expected CreateAccessToken to be called",
					)
				}

				if !tt.token.createRefreshCalled {
					t.Fatal(
						"expected CreateRefreshToken to be called",
					)
				}

				if tt.token.accessUserID != 1 {
					t.Errorf(
						"expected access token user ID 1, got %d",
						tt.token.accessUserID,
					)
				}

				if tt.token.refreshUserID != 1 {
					t.Errorf(
						"expected refresh token user ID 1, got %d",
						tt.token.refreshUserID,
					)
				}

				found := false

				for _, cookie := range w.Result().Cookies() {
					if cookie.Name != "refresh_token" {
						continue
					}

					found = true

					if !cookie.HttpOnly {
						t.Fatal(
							"refresh_token cookie must be HttpOnly",
						)
					}

					if cookie.Value != "refresh-token" {
						t.Errorf(
							"expected refresh-token cookie value, got %q",
							cookie.Value,
						)
					}
				}

				if !found {
					t.Fatal(
						"refresh_token cookie was not set",
					)
				}

				if !strings.Contains(
					w.Body.String(),
					`"jwt"`,
				) {
					t.Fatalf(
						"response does not contain jwt: %s",
						w.Body.String(),
					)
				}
			},
		)
	}
}

func TestHandler_CreateUser(t *testing.T) {
	internalErr := errors.New(
		"insert failed",
	)

	tests := []struct {
		name       string
		body       string
		auth       *fakeAuthService
		wantStatus int
	}{
		{
			name: "ok: user created",
			body: `{
				"email":"ada@example.com",
				"password":"correcthorse"
			}`,
			auth:       &fakeAuthService{},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "bad request: malformed json",
			body:       `{not-json`,
			auth:       &fakeAuthService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation: short password",
			body: `{
				"email":"ada@example.com",
				"password":"short"
			}`,
			auth:       &fakeAuthService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "unknown service error maps to 500",
			body: `{
				"email":"ada@example.com",
				"password":"correcthorse"
			}`,
			auth: &fakeAuthService{
				saveUserErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				h := newTestHandler(
					tt.auth,
					nil,
					nil,
					nil,
					&fakeTokenService{},
				)

				req := httptest.NewRequest(
					http.MethodPost,
					"/signup",
					strings.NewReader(tt.body),
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

				if tt.wantStatus != http.StatusCreated {
					return
				}

				if !tt.auth.saveUserCalled {
					t.Fatal(
						"expected SaveUser to be called",
					)
				}

				if tt.auth.savedEmail != "ada@example.com" {
					t.Errorf(
						"expected email ada@example.com, got %q",
						tt.auth.savedEmail,
					)
				}

				if string(
					tt.auth.savedPassword,
				) != "correcthorse" {
					t.Errorf(
						"expected password correcthorse, got %q",
						string(
							tt.auth.savedPassword,
						),
					)
				}
			},
		)
	}
}

func TestHandler_IssueJWT(t *testing.T) {
	internalErr := errors.New(
		"token signing failed",
	)

	tests := []struct {
		name       string
		cookie     *http.Cookie
		token      *fakeTokenService
		wantStatus int
	}{
		{
			name: "ok: valid refresh token",
			cookie: &http.Cookie{
				Name:  "refresh_token",
				Value: "old-refresh-token",
			},
			token: &fakeTokenService{
				rotateUserID: 1,
				rotatedToken: "new-refresh-token",
				rotateExpiresAt: time.Now().Add(
					30 * 24 * time.Hour,
				),
				accessToken: "new-access-token",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized: missing refresh token",
			token:      &fakeTokenService{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unauthorized: empty refresh token",
			cookie: &http.Cookie{
				Name:  "refresh_token",
				Value: "",
			},
			token:      &fakeTokenService{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unauthorized: invalid refresh token",
			cookie: &http.Cookie{
				Name:  "refresh_token",
				Value: "invalid",
			},
			token: &fakeTokenService{
				rotateErr: errors.New(
					"expired",
				),
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "access token creation error maps to 500",
			cookie: &http.Cookie{
				Name:  "refresh_token",
				Value: "old-refresh-token",
			},
			token: &fakeTokenService{
				rotateUserID: 1,
				rotatedToken: "new-refresh-token",
				rotateExpiresAt: time.Now().Add(
					30 * 24 * time.Hour,
				),
				accessTokenErr: internalErr,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				h := newTestHandler(
					&fakeAuthService{},
					nil,
					nil,
					nil,
					tt.token,
				)

				req := httptest.NewRequest(
					http.MethodGet,
					"/token",
					nil,
				)

				if tt.cookie != nil {
					req.AddCookie(
						tt.cookie,
					)
				}

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

				if tt.wantStatus != http.StatusOK {
					return
				}

				if !tt.token.rotateCalled {
					t.Fatal(
						"expected RotateRefreshToken to be called",
					)
				}

				if tt.token.oldRefreshToken != "old-refresh-token" {
					t.Errorf(
						"expected old refresh token %q, got %q",
						"old-refresh-token",
						tt.token.oldRefreshToken,
					)
				}

				if !tt.token.createAccessCalled {
					t.Fatal(
						"expected CreateAccessToken to be called",
					)
				}

				if tt.token.accessUserID != 1 {
					t.Errorf(
						"expected access token user ID 1, got %d",
						tt.token.accessUserID,
					)
				}

				found := false

				for _, cookie := range w.Result().Cookies() {
					if cookie.Name != "refresh_token" {
						continue
					}

					found = true

					if !cookie.HttpOnly {
						t.Fatal(
							"refresh_token cookie must be HttpOnly",
						)
					}

					if cookie.Value != "new-refresh-token" {
						t.Errorf(
							"expected new refresh token, got %q",
							cookie.Value,
						)
					}
				}

				if !found {
					t.Fatal(
						"new refresh token cookie was not set",
					)
				}

				if !strings.Contains(
					w.Body.String(),
					`"jwt"`,
				) {
					t.Fatalf(
						"response does not contain jwt: %s",
						w.Body.String(),
					)
				}
			},
		)
	}
}
