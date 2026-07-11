// Package utils provides miscellaneous helpers for system operations.
package utils

import (
	"fmt"
	"strings"
)

// BuildShortURL constructs the absolute shortened URL using the config BASE_URL and slug code.
func BuildShortURL(baseURL, shortCode string) string {
	return fmt.Sprintf("%s/r/%s", strings.TrimSuffix(baseURL, "/"), shortCode)
}
