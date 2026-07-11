// Package utils provides common helper functions.
package utils

import (
	"net/url"
	"strings"
)

// IsValidURL validates if a string is a well-formed HTTP/HTTPS URL.
func IsValidURL(toTest string) bool {
	u, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" || !strings.Contains(u.Host, ".") {
		return false
	}
	return true
}
