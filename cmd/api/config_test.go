package main

import (
	"strings"
	"testing"
)

func TestValidateJWTSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{name: "empty", secret: "", wantErr: true},
		{name: "too short", secret: "supersecret", wantErr: true},
		{name: "31 bytes", secret: strings.Repeat("a", 31), wantErr: true},
		{name: "32 bytes", secret: strings.Repeat("a", 32), wantErr: false},
		{name: "64 bytes", secret: strings.Repeat("a", 64), wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateJWTSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), tt.secret) && tt.secret != "" {
				t.Error("error message must not contain the secret itself")
			}
		})
	}
}
