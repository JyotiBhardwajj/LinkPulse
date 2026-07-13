// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a middleware that logs incoming HTTP requests using slog.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		if path == "/metrics" || path == "/health" || path == "/ready" || (len(path) >= 8 && path[:8] == "/health/") {
			c.Next()
			return
		}

		c.Next()

		latencyMS := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		reqID := GetRequestID(c)

		fullPath := path
		if query != "" {
			fullPath = path + "?" + query
		}

		// Choose appropriate log level depending on status code
		var logFn func(string, ...any)
		if status >= 500 {
			logFn = slog.Error
		} else if status >= 400 {
			logFn = slog.Warn
		} else {
			logFn = slog.Info
		}

		logArgs := []any{
			slog.String("request_id", reqID),
			slog.String("method", c.Request.Method),
			slog.String("path", fullPath),
			slog.Int("status", status),
			slog.Int64("latency_ms", latencyMS),
			slog.String("client_ip", c.ClientIP()),
		}

		if authCtx, exists := GetAuthContext(c); exists {
			logArgs = append(logArgs, slog.String("user_id", authCtx.UserID.String()))
		}

		logFn("HTTP request processed", logArgs...)
	}
}
