// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"context"

	"linkpulse/internal/constants"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID returns a middleware that injects a unique request ID into the request headers and context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Header("X-Request-ID", reqID)
		// Store in Gin context
		c.Set(string(constants.RequestIDKey), reqID)
		// Store in standard Go context
		ctx := context.WithValue(c.Request.Context(), constants.RequestIDKey, reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetRequestID extracts the request ID from the Gin context.
func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get(string(constants.RequestIDKey)); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}
