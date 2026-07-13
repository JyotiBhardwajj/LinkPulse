// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// Deprecated injects deprecation headers to let clients know an endpoint is scheduled for decommissioning.
func Deprecated(deprecationDate string, sunsetDate string, successorURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deprecationDate != "" {
			c.Header("Deprecation", fmt.Sprintf("\"%s\"", deprecationDate))
		}
		if sunsetDate != "" {
			c.Header("Sunset", sunsetDate)
		}
		if successorURL != "" {
			c.Header("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", successorURL))
		}
		c.Next()
	}
}
