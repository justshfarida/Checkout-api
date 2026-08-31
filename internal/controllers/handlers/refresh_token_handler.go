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

// RefreshToken exchanges a valid refresh token for a new token pair.
// @Summary Refresh an access token
// @Description Exchanges a valid refresh token (sent via header) for a new access/refresh token pair
// @Tags auth
// @Produce json
// @Param Refresh-Token header string true "Refresh token"
// @Success 200 {object} dtos.LoginResponse
// @Failure 400 {object} httperror.Envelope
// @Failure 401 {object} httperror.Envelope
// @Router /token [post]
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
