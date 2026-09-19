package repositories

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")

	ErrOrderNotFound   = errors.New("order not found")
	ErrOrderNotPending = errors.New("order is not pending")
)
