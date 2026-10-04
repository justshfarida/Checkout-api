package handlers

import (
	"checkout-api/internal/middlewares"
	"checkout-api/internal/services"
	"checkout-api/internal/validation"
	"encoding/json"
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

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Invalid request body",
		)
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

		handleError(w, err)
		return
	}

	accessToken, err := h.tokenService.CreateAccessToken(
		user.ID,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	refreshToken,
		refreshExpiresAt,
		err := h.tokenService.CreateRefreshToken(
		r.Context(),
		user.ID,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			Expires:  refreshExpiresAt,
			MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		},
	)

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

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Invalid request body",
		)
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
		handleError(w, err)
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
		handleError(w, err)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    newRefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			Expires:  newRefreshExpiresAt,
			MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		},
	)

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

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		},
	)

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
		handleError(w, err)
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
		handleError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		inventory,
	)
}
