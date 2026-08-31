package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

func ClampLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

type OffsetPage struct {
	Limit  int
	Offset int
}

func ParseOffsetPage(r *http.Request) OffsetPage {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return OffsetPage{Limit: ClampLimit(limit), Offset: offset}
}
