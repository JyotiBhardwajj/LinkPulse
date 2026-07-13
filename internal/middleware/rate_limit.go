// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"github.com/gin-gonic/gin"
)

// RateLimit acts as a placeholder for token-bucket or sliding-window rate limiting.
// Injects standard Rate Limiting headers (X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After).
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Default limits (100 requests per window placeholder)
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "99")
		c.Header("Retry-After", "0")

		c.Next()
	}
}
