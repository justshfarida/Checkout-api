package handlers

import (
	"checkout-api/internal/models"
	"context"
)

type IdempotencyService interface {
	SaveIdempotencyRecord(ctx context.Context, key, requestHash string, statusCode int, body any) error
	GetIdempotencyRecord(ctx context.Context, key string) (*models.IdempotencyRecord, error)
}

type IdempotencyHandler struct {
	service IdempotencyService
}

func NewIdempotencyHandler(service IdempotencyService) *IdempotencyHandler {
	return &IdempotencyHandler{service: service}
}
