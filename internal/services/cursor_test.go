package services

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr error
	}{
		{name: "empty cursor means first page", raw: "", want: 0},
		{name: "round-trips an encoded id", raw: EncodeCursor(41), want: 41},
		{name: "not base64 is invalid", raw: "***", wantErr: ErrInvalidCursor},
		{name: "base64 but not JSON is invalid", raw: base64.RawURLEncoding.EncodeToString([]byte("hello")), wantErr: ErrInvalidCursor},
		{name: "negative id is invalid", raw: base64.RawURLEncoding.EncodeToString([]byte(`{"id":-1}`)), wantErr: ErrInvalidCursor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeCursor(tt.raw)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DecodeCursor() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("DecodeCursor() = %d, want %d", got, tt.want)
			}
		})
	}
}
