// Package utils provides common helper functions.
package utils

import (
	"crypto/rand"
	"math/big"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenerateBase62Code creates a random cryptographically secure Base62 string of the specified length.
func GenerateBase62Code(length int) (string, error) {
	result := make([]byte, length)
	charsLen := big.NewInt(int64(len(base62Chars)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsLen)
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[num.Int64()]
	}

	return string(result), nil
}
