// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// Recovery recovers from panics and logs them via slog.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID := GetRequestID(c)
				slog.Error("Panic recovered",
					slog.String("request_id", reqID),
					slog.Any("error", err),
					slog.String("stack", string(debug.Stack())),
				)

				utils.SendError(c, http.StatusInternalServerError, "Internal server error", "PANIC_RECOVERED")
				c.Abort()
			}
		}()
		c.Next()
	}
}
