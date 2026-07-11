// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"github.com/gin-gonic/gin"
)

// Auth acts as a placeholder for JWT-based user authentication.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder: Future phase will parse JWT and store user UUID in Context.
		c.Next()
	}
}
