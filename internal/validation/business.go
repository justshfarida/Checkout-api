package validation

import (
	"errors"
	"fmt"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
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
