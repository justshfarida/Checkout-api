package handlers

import (
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/models"
	"context"
	"net/http"
	"time"
)

type RefreshTokenService interface {
	SaveRefreshToken(ctx context.Context, token string, userID int, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

type RefreshTokenHandler struct {
	service RefreshTokenService
}

func NewRefreshTokenHandler(service RefreshTokenService) *RefreshTokenHandler {
	return &RefreshTokenHandler{service: service}
}

func (handler *RefreshTokenHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	oldToken := r.Header.Get("Refresh-Token")
	if oldToken == "" {
		http.Error(w, "missing Refresh-Token header", http.StatusBadRequest)
		return
	}

	rt, err := handler.service.GetRefreshToken(ctx, oldToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	if rt.Revoked {
		http.Error(w, "refresh token has been revoked", http.StatusUnauthorized)
		return
	}
	if time.Now().After(rt.ExpiresAt) {
		http.Error(w, "refresh token expired", http.StatusUnauthorized)
		return
	}

	if err := handler.service.RevokeRefreshToken(ctx, oldToken); err != nil {
		http.Error(w, "failed to rotate token", http.StatusInternalServerError)
		return
	}

	newAccessToken, err := handler.authManager.GenerateAccessToken(rt.UserID)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}
	newRefreshToken, err := handler.authManager.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}
	if err := handler.service.SaveRefreshToken(ctx, newRefreshToken, rt.UserID, time.Now().Add(30*24*time.Hour)); err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, dtos.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	})
}
