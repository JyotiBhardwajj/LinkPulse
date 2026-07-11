// Package auth handles authentication operations, JWT generation, and RTR.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateSecureToken creates a cryptographically secure 32-byte hex-encoded string.
func GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
