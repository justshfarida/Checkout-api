package validation

import (
	"errors"
	"fmt"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrEmptyOrder        = errors.New("order must contain at least one item")
)

func ValidateStock(
	stock int,
	quantity int,
) error {
	if quantity > stock {
		return fmt.Errorf(
			"%w: available=%d requested=%d",
			ErrInsufficientStock,
			stock,
			quantity,
		)
	}

	return nil
}

func ValidateOrderItemsCount(
	count int,
) error {
	if count == 0 {
		return ErrEmptyOrder
	}

	return nil
}
