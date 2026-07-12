package middleware

import (
	"strconv"
	"time"

	"linkpulse/internal/metrics"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware records request count, status, and duration using static path routes.
func MetricsMiddleware(tracker metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		// Retrieve matching Gin route template to protect cardinality
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		duration := time.Since(start)
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		tracker.RecordHTTPRequest(method, route, status)
		tracker.RecordRequestDuration(method, route, duration)
	}
}
