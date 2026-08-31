package handlers

import (
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/models"
	"checkout-api/internal/validation"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type UserService interface {
	SignUp(ctx context.Context, username string, password string) (*models.User, error)
	Login(ctx context.Context, username, password string) (*models.User, error)
}

type RefreshTokenSaver interface {
	SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error
}

type AuthManager interface {
	GenerateAccessToken(userID int) (string, error)
	GenerateRefreshToken() (string, error)
}

type UserHandler struct {
	service        UserService
	refreshService RefreshTokenSaver
	authManager    AuthManager
}

func NewUserHandler(service UserService, refreshService RefreshTokenSaver, authManager AuthManager) *UserHandler {
	return &UserHandler{service: service, refreshService: refreshService, authManager: authManager}
}

// SignUp registers a new user.
// @Summary Create a new user account
// @Description Registers a new user with a username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.SignUpRequest true "New user details"
// @Success 201 {object} models.User
// @Failure 400 {object} httperror.Envelope
// @Router /signup [post]
func (handler *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dtos.SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "invalid request body")
		return
	}

	req.Username = validation.SanitizeUsername(req.Username)

	result := validation.NewResult()
	result.Required("username", req.Username)
	result.MinLen("username", req.Username, 3)
	result.MaxLen("username", req.Username, 32)
	result.Required("password", req.Password)
	result.MinLen("password", req.Password, 8)
	if !result.Valid() {
		WriteShapeErrors(w, result)
		return
	}

	user, err := handler.service.SignUp(ctx, req.Username, req.Password)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// Login authenticates a user and issues a token pair.
// @Summary Log in
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.LoginRequest true "Credentials"
// @Success 200 {object} dtos.LoginResponse
// @Failure 401 {object} httperror.Envelope
// @Router /login [post]
func (handler *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dtos.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "invalid request body")
		return
	}

	req.Username = validation.SanitizeUsername(req.Username)

	result := validation.NewResult()
	result.Required("username", req.Username)
	result.Required("password", req.Password)
	if !result.Valid() {
		WriteShapeErrors(w, result)
		return
	}

	user, err := handler.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	accessToken, err := handler.authManager.GenerateAccessToken(user.ID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	refreshToken, err := handler.authManager.GenerateRefreshToken()
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	if err := handler.refreshService.SaveRefreshToken(ctx, refreshToken, user.ID, time.Now().Add(30*24*time.Hour)); err != nil {
		WriteServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dtos.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}
