package webhooklab

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SignatureHeader carries the hex HMAC-SHA256 of the raw request body.
const SignatureHeader = "X-Signature"

// Sign returns hex(HMAC-SHA256(secret, body)). The sender and the receiver
// share the secret; only someone who knows it can produce a valid signature.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether signature is the correct HMAC of body. An empty
// signature is always invalid. A "sha256=" prefix (GitHub style) is accepted.
//
// hmac.Equal compares in constant time: a plain == stops at the first wrong
// byte, so an attacker could guess the signature byte by byte by measuring
// how long the comparison takes (timing attack).
func Verify(secret string, body []byte, signature string) bool {
	signature = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(signature), "sha256="))
	if signature == "" {
		return false
	}
	expected := Sign(secret, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}
