package pagination

type PageResult[T any] struct {
	Items      []T    `json:"items"`
	Total      int    `json:"total,omitempty"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
