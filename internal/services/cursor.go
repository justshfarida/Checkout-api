package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type cursorPayload struct {
	LastID int `json:"id"`
}

func EncodeCursor(lastID int) string {
	b, _ := json.Marshal(
		cursorPayload{
			LastID: lastID,
		},
	)

	return base64.
		RawURLEncoding.
		EncodeToString(b)
}

func DecodeCursor(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	b, err :=
		base64.RawURLEncoding.
			DecodeString(raw)

	if err != nil {
		return 0, fmt.Errorf(
			"%w: not valid base64",
			ErrInvalidCursor,
		)
	}

	var cursor cursorPayload

	if err := json.Unmarshal(
		b,
		&cursor,
	); err != nil {
		return 0, fmt.Errorf(
			"%w: invalid payload",
			ErrInvalidCursor,
		)
	}

	if cursor.LastID < 0 {
		return 0, fmt.Errorf(
			"%w: negative id",
			ErrInvalidCursor,
		)
	}

	return cursor.LastID, nil
}
