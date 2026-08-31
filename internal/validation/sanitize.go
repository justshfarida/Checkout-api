package validation

import "strings"

func SanitizeString(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func SanitizeUsername(s string) string {
	return strings.ToLower(SanitizeString(s))
}
