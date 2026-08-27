package handlers

import (
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	SignUp(ctx context.Context, username string, password string) (*models.User, error)
	Login(ctx context.Context, username, password string) (*models.User, error)
}

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (handler *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dtos.SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to process password", http.StatusInternalServerError)
		return
	}

	user, err := handler.service.SignUp(ctx, req.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (handler *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dtos.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	user, err := handler.store.Login(ctx, req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	accessToken, err := handler.authManager.GenerateAccessToken(user.ID)

	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	refreshToken, err := handler.authManager.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	if err := handler.store.SaveRefreshToken(ctx, refreshToken, user.ID, time.Now().Add(30*24*time.Hour)); err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, dtos.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}
