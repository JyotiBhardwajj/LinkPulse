// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"context"
	"net/http"
	"time"

	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// Timeout attaches a timeout to the request context.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	// If duration is 0 or less, skip timeout logic.
	if timeout <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{}, 1)
		panicChan := make(chan interface{}, 1)

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicChan <- p
				}
			}()
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			return
		case p := <-panicChan:
			panic(p)
		case <-ctx.Done():
			utils.SendError(c, http.StatusGatewayTimeout, "Request timeout exceeded", "TIMEOUT")
			c.Abort()
		}
	}
}
