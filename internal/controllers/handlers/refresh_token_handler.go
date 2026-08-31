package handlers

import (
	"checkout-api/internal/controllers/dtos"
	"checkout-api/internal/service"
	"context"
	"net/http"
)

type RefreshTokenService interface {
	Rotate(ctx context.Context, oldToken string) (*service.TokenPair, error)
}

type RefreshTokenHandler struct {
	service RefreshTokenService
}

func NewRefreshTokenHandler(service RefreshTokenService) *RefreshTokenHandler {
	return &RefreshTokenHandler{service: service}
}

func (handler *RefreshTokenHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	oldToken := r.Header.Get("Refresh-Token")
	if oldToken == "" {
		WriteValidationError(w, "missing Refresh-Token header")
		return
	}

	pair, err := handler.service.Rotate(r.Context(), oldToken)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dtos.LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}
