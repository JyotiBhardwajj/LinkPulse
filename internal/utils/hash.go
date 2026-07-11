// Package utils provides common helper functions.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashIP generates a SHA-256 hash of the provided IP address string for anonymity.
func HashIP(ip string) string {
	if ip == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}

// HashSHA256 returns a hex-encoded SHA-256 checksum of the input string.
func HashSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
