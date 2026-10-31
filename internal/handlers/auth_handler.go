package handlers

import (
	"checkout-api/internal/middlewares"
	"checkout-api/internal/services"
	"checkout-api/internal/validation"
	"errors"
	"net/http"
	"time"
)

// LoginUser godoc
// @Summary Login user
// @Description Authenticates a user and returns an access token. A refresh token is stored in an HttpOnly cookie.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body AuthRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /login [post]
func (h *Handler) LoginUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AuthRequest

	if !decodeJSON(w, r, &req) {
		return
	}

	req.Email = validation.SanitizeEmail(
		req.Email,
	)

	if err := h.validate.Struct(req); err != nil {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"VALIDATION_FAILED",
			"Request validation failed",
			validationDetails(err)...,
		)
		return
	}

	user, err := h.authService.Authenticate(
		r.Context(),
		req.Email,
		[]byte(req.Password),
	)

	if err != nil {
		if errors.Is(
			err,
			services.ErrInvalidCredentials,
		) {
			writeError(
				w,
				http.StatusUnauthorized,
				"INVALID_CREDENTIALS",
				"Email or password is incorrect",
			)
			return
		}

		handleError(w, r, err)
		return
	}

	accessToken, err := h.tokenService.CreateAccessToken(
		user.ID,
	)

	if err != nil {
		handleError(w, r, err)
		return
	}

	refreshToken,
		refreshExpiresAt,
		err := h.tokenService.CreateRefreshToken(
		r.Context(),
		user.ID,
	)

	if err != nil {
		handleError(w, r, err)
		return
	}

	h.setRefreshCookie(w, refreshToken, refreshExpiresAt, int((7 * 24 * time.Hour).Seconds()))

	writeJSON(
		w,
		http.StatusOK,
		AuthResponse{
			JWT: accessToken,
		},
	)
}

// CreateUser godoc
// @Summary Create user
// @Description Registers a new user.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body AuthRequest true "User registration data"
// @Success 201 "User created successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /signup [post]
func (h *Handler) CreateUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AuthRequest

	if !decodeJSON(w, r, &req) {
		return
	}

	req.Email = validation.SanitizeEmail(
		req.Email,
	)

	if err := h.validate.Struct(req); err != nil {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"VALIDATION_FAILED",
			"Request validation failed",
			validationDetails(err)...,
		)
		return
	}

	err := h.authService.SaveUser(
		r.Context(),
		req.Email,
		[]byte(req.Password),
	)

	if err != nil {
		handleError(w, r, err)
		return
	}

	w.WriteHeader(
		http.StatusCreated,
	)
}

// IssueJWT godoc
// @Summary Refresh access token
// @Description Rotates the refresh token stored in the HttpOnly cookie and returns a new access token.
// @Tags Auth
// @Produce json
// @Success 200 {object} AuthResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /token [post]
func (h *Handler) IssueJWT(
	w http.ResponseWriter,
	r *http.Request,
) {
	cookie, err := r.Cookie(
		"refresh_token",
	)

	if err != nil {
		if errors.Is(
			err,
			http.ErrNoCookie,
		) {
			writeError(
				w,
				http.StatusUnauthorized,
				"REFRESH_TOKEN_REQUIRED",
				"Refresh token is missing",
			)
			return
		}

		writeError(
			w,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication failed",
		)
		return
	}

	oldRefreshToken := cookie.Value

	if oldRefreshToken == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			"REFRESH_TOKEN_REQUIRED",
			"Refresh token is missing",
		)
		return
	}

	userID,
		newRefreshToken,
		newRefreshExpiresAt,
		err := h.tokenService.RotateRefreshToken(
		r.Context(),
		oldRefreshToken,
	)

	if err != nil {
		writeError(
			w,
			http.StatusUnauthorized,
			"INVALID_REFRESH_TOKEN",
			"Refresh token is invalid or expired",
		)
		return
	}

	accessToken, err := h.tokenService.CreateAccessToken(
		userID,
	)

	if err != nil {
		handleError(w, r, err)
		return
	}

	h.setRefreshCookie(w, newRefreshToken, newRefreshExpiresAt, int((7 * 24 * time.Hour).Seconds()))

	writeJSON(
		w,
		http.StatusOK,
		AuthResponse{
			JWT: accessToken,
		},
	)
}

func (h *Handler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	cookie, err := r.Cookie("refresh_token")

	if err == nil {
		if err := h.tokenService.Logout(
			r.Context(),
			cookie.Value,
		); err != nil {
			writeError(
				w,
				http.StatusInternalServerError,
				"LOGOUT_FAILED",
				"failed to log out",
			)
			return
		}
	}

	h.setRefreshCookie(w, "", time.Unix(0, 0), -1)

	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) GetCurrentUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := middlewares.UserIDFromContext(
		r.Context(),
	)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication is required",
		)
		return
	}

	user, err := h.authService.GetUserByID(
		r.Context(),
		userID,
	)
	if err != nil {
		handleError(w, r, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]any{
			"id":      user.ID,
			"email":   user.Email,
			"balance": user.Balance,
		},
	)
}
func (h *Handler) GetUserInventory(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := middlewares.UserIDFromContext(
		r.Context(),
	)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication is required",
		)
		return
	}

	inventory, err := h.authService.GetUserInventory(
		r.Context(),
		userID,
	)
	if err != nil {
		handleError(w, r, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		inventory,
	)
}

// setRefreshCookie writes the refresh-token cookie. HttpOnly keeps it away
// from JavaScript (XSS), SameSite=Lax blocks cross-site POSTs (CSRF), and
// Secure keeps it off plain HTTP unless COOKIE_SECURE=false (local dev).
func (h *Handler) setRefreshCookie(
	w http.ResponseWriter,
	value string,
	expires time.Time,
	maxAge int,
) {
	// #nosec G124 -- Secure is h.secureCookies: true by default, false only when
	// COOKIE_SECURE=false for local HTTP; HttpOnly and SameSite are always set.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   maxAge,
	})
}
