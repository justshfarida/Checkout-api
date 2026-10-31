package services

import "errors"

var (
	ErrItemNotFound       = errors.New("item not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCursor      = errors.New("invalid cursor")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrOrderNotFound      = errors.New("order not found")
	ErrEmptyCart          = errors.New("The cart is empty")

	// ErrPaymentProviderUnavailable means the payment provider is failing and
	// the circuit breaker is rejecting calls to it for now.
	ErrPaymentProviderUnavailable = errors.New("payment provider unavailable")
	ErrPasswordHashing            = errors.New("failed to hash password")
)
