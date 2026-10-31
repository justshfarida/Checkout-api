package main

import "fmt"

// minJWTSecretBytes: HS256 needs a key at least as long as its 256-bit
// output. A shorter or guessable secret lets an attacker brute-force it
// offline from one token and then sign tokens for any user.
const minJWTSecretBytes = 32

// validateJWTSecret makes the app refuse to start with a weak secret
// (fail fast) instead of running with one.
func validateJWTSecret(secret string) error {
	if len(secret) < minJWTSecretBytes {
		return fmt.Errorf(
			"JWT_SECRET must be at least %d bytes, got %d",
			minJWTSecretBytes,
			len(secret),
		)
	}
	return nil
}
