// Package utils provides common helper functions.
package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ComputeETag computes a deterministic SHA-256 hash of the marshaled JSON representation of a resource.
func ComputeETag(data interface{}) string {
	if data == nil {
		return ""
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(bytes)
	return fmt.Sprintf("W/\"%x\"", hash)
}

// CheckIfNoneMatch checks If-None-Match header and returns true if it matches computed ETag.
// Safe methods strictly only (GET, HEAD).
func CheckIfNoneMatch(c *gin.Context, computedETag string) bool {
	if computedETag == "" {
		return false
	}

	// Only return 304 Not Modified for safe methods (GET, HEAD)
	method := c.Request.Method
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}

	clientETag := c.GetHeader("If-None-Match")
	if clientETag == "" {
		return false
	}

	return clientETag == computedETag
}
