package service

import "errors"

var (
	ErrQuantityLimitExceeded = errors.New("quantity limit exceeded")
	ErrItemNotFound          = errors.New("item not found")
	ErrInsufficientStock     = errors.New("insufficient stock")
	ErrCartItemNotFound      = errors.New("cart item not found")
	ErrUsernameTaken         = errors.New("username already taken")
	ErrInvalidCredentials    = errors.New("invalid username or password")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrCartEmpty             = errors.New("cart is empty or does not exist")
	ErrInvalidQuantity       = errors.New("invalid quantity")
	ErrIdempotencyConflict   = errors.New("idempotency key already used with a different request body")
	ErrRefreshTokenRevoked   = errors.New("refresh token has been revoked")
	ErrRefreshTokenExpired   = errors.New("refresh token has expired")
)
