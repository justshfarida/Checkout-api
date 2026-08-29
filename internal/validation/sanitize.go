package validation

import "strings"

func SanitizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
