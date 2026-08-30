package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"checkout-api/internal/validation"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
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

	// SANITIZE
	req.Email =
		validation.SanitizeEmail(
			req.Email,
		)

	// SHAPE VALIDATION
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

	user, err :=
		h.authService.FindUserByEmail(
			r.Context(),
			req.Email,
		)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(
				w,
				http.StatusUnauthorized,
				"INVALID_CREDENTIALS",
				"Email or password is incorrect",
			)
			return
		}

		handleError(
			w,
			err,
		)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		user.Hash,
		[]byte(req.Password),
	); err != nil {
		writeError(
			w,
			http.StatusUnauthorized,
			"INVALID_CREDENTIALS",
			"Email or password is incorrect",
		)
		return
	}

	accessExpiresAt :=
		time.Now().Add(
			15 * time.Minute,
		)

	token :=
		jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					accessExpiresAt,
				),
				Subject: strconv.Itoa(
					user.ID,
				),
				IssuedAt: jwt.NewNumericDate(
					time.Now(),
				),
			},
		)

	signedJWT, err :=
		token.SignedString(
			h.jwtSecret,
		)

	if err != nil {
		handleError(
			w,
			fmt.Errorf(
				"failed to sign access token: %w",
				err,
			),
		)
		return
	}

	refreshToken, err :=
		generateRefreshToken()

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	refreshTokenHash :=
		hashRefreshToken(
			refreshToken,
		)

	refreshTokenExpiresAt :=
		time.Now().Add(
			30 * 24 * time.Hour,
		)

	err =
		h.authService.SaveRefreshToken(
			r.Context(),
			user.ID,
			refreshTokenHash,
			refreshTokenExpiresAt,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
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
			Expires:  refreshTokenExpiresAt,
			MaxAge:   30 * 24 * 60 * 60,
		},
	)

	writeJSON(
		w,
		http.StatusOK,
		AuthResponse{
			JWT: signedJWT,
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

	// SANITIZE
	req.Email =
		validation.SanitizeEmail(
			req.Email,
		)

	// SHAPE VALIDATION
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

	hash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)

	if err != nil {
		handleError(
			w,
			fmt.Errorf(
				"failed to hash password: %w",
				err,
			),
		)
		return
	}

	err =
		h.authService.SaveUser(
			r.Context(),
			req.Email,
			hash,
		)

	if err != nil {
		handleError(
			w,
			err,
		)
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
	cookie, err :=
		r.Cookie(
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

	oldRefreshToken :=
		cookie.Value

	if oldRefreshToken == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			"REFRESH_TOKEN_REQUIRED",
			"Refresh token is missing",
		)
		return
	}

	newRefreshToken, err :=
		generateRefreshToken()

	if err != nil {
		handleError(
			w,
			err,
		)
		return
	}

	oldTokenHash :=
		hashRefreshToken(
			oldRefreshToken,
		)

	newTokenHash :=
		hashRefreshToken(
			newRefreshToken,
		)

	newRefreshExpiresAt :=
		time.Now().Add(
			30 * 24 * time.Hour,
		)

	userID, err :=
		h.authService.RotateRefreshToken(
			r.Context(),
			oldTokenHash,
			newTokenHash,
			newRefreshExpiresAt,
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

	accessExpiresAt :=
		time.Now().Add(
			5 * time.Minute,
		)

	token :=
		jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.RegisteredClaims{
				Subject: strconv.Itoa(
					userID,
				),
				IssuedAt: jwt.NewNumericDate(
					time.Now(),
				),
				ExpiresAt: jwt.NewNumericDate(
					accessExpiresAt,
				),
			},
		)

	signedJWT, err :=
		token.SignedString(
			h.jwtSecret,
		)

	if err != nil {
		handleError(
			w,
			fmt.Errorf(
				"failed to sign access token: %w",
				err,
			),
		)
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
			MaxAge:   30 * 24 * 60 * 60,
		},
	)

	writeJSON(
		w,
		http.StatusOK,
		AuthResponse{
			JWT: signedJWT,
		},
	)
}

func generateRefreshToken() (
	string,
	error,
) {
	randomBytes :=
		make(
			[]byte,
			32,
		)

	if _, err :=
		rand.Read(
			randomBytes,
		); err != nil {

		return "",
			fmt.Errorf(
				"failed to generate refresh token: %w",
				err,
			)
	}

	return base64.
			RawURLEncoding.
			EncodeToString(
				randomBytes,
			),
		nil
}

func hashRefreshToken(
	token string,
) []byte {
	hash :=
		sha256.Sum256(
			[]byte(token),
		)

	return hash[:]
}
