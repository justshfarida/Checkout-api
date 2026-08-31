package pagination

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
)

type CursorPage struct {
	Limit int
	After int
}

func ParseCursorPage(r *http.Request) (CursorPage, error) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	page := CursorPage{Limit: ClampLimit(limit)}

	cursor := query.Get("cursor")
	if cursor == "" {
		return page, nil
	}
	after, err := DecodeCursor(cursor)
	if err != nil {
		return CursorPage{}, fmt.Errorf("invalid cursor: %w", err)
	}
	page.After = after
	return page, nil
}

func EncodeCursor(lastID int) string {
	return base64.URLEncoding.EncodeToString([]byte(strconv.Itoa(lastID)))
}

func DecodeCursor(cursor string) (int, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(string(raw))
	if err != nil {
		return 0, err
	}
	return id, nil
}
