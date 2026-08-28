package services

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}

	if limit > MaxLimit {
		return MaxLimit
	}

	return limit
}
