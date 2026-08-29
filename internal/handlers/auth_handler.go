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

func (h *Handler) LoginUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AuthRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(
			w,
			"invalid body",
			http.StatusBadRequest,
		)
		return
	}

	req.Email =
		validation.SanitizeEmail(
			req.Email,
		)

	if err := h.validate.Struct(req); err != nil {
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			ErrorMessageResponse{
				Message: err.Error(),
			},
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
			writeJSON(
				w,
				http.StatusUnprocessableEntity,
				ErrorMessageResponse{
					Message: "user does not exist",
				},
			)
			return
		}

		fmt.Printf(
			"cannot query user: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		user.Hash,
		[]byte(req.Password),
	); err != nil {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	accessExpiresAt :=
		time.Now().Add(
			15 * time.Minute,
		)

	token := jwt.NewWithClaims(
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
		fmt.Printf(
			"cannot generate signed JWT: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	refreshToken, err :=
		generateRefreshToken()

	if err != nil {
		fmt.Printf(
			"cannot generate refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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
		fmt.Printf(
			"cannot save refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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

func (h *Handler) CreateUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AuthRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(
			w,
			"invalid body",
			http.StatusBadRequest,
		)
		return
	}

	// 1. SANITIZE
	req.Email =
		validation.SanitizeEmail(
			req.Email,
		)

	// 2. SHAPE VALIDATION
	if err := h.validate.Struct(req); err != nil {
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			ErrorMessageResponse{
				Message: err.Error(),
			},
		)
		return
	}

	hash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)

	if err != nil {
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		nil,
	)
}

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
			http.Error(
				w,
				"Refresh token is missing",
				http.StatusUnauthorized,
			)
			return
		}

		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	oldRefreshToken :=
		cookie.Value

	if oldRefreshToken == "" {
		http.Error(
			w,
			"Refresh token is missing",
			http.StatusUnauthorized,
		)
		return
	}

	newRefreshToken, err :=
		generateRefreshToken()

	if err != nil {
		fmt.Printf(
			"failed to generate refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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
		fmt.Printf(
			"failed to rotate refresh token: %v\n",
			err,
		)

		http.Error(
			w,
			"Invalid or expired refresh token",
			http.StatusUnauthorized,
		)
		return
	}

	accessExpiresAt :=
		time.Now().Add(
			5 * time.Minute,
		)

	token := jwt.NewWithClaims(
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
		fmt.Printf(
			"failed to sign JWT: %v\n",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
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
