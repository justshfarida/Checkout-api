package services

import "errors"

var (
	ErrItemNotFound  = errors.New("item not found")
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidCursor = errors.New("invalid cursor")
)
