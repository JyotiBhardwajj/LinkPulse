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

		c.Next()

		latency := time.Since(start)
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

		logFn("HTTP request processed",
			slog.String("request_id", reqID),
			slog.String("method", c.Request.Method),
			slog.String("path", fullPath),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		)
	}
}
