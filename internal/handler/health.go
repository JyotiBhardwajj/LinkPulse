package handler

import (
	"net/http"
	"time"

	"linkpulse/internal/health"

	"github.com/gin-gonic/gin"
)

// HealthHandler manages application diagnostic checks.
type HealthHandler struct {
	healthSvc   health.HealthService
	version     string
	gitCommit   string
	buildTime   string
	environment string
	startTime   time.Time
}

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler(healthSvc health.HealthService, version, gitCommit, buildTime, environment string) *HealthHandler {
	return &HealthHandler{
		healthSvc:   healthSvc,
		version:     version,
		gitCommit:   gitCommit,
		buildTime:   buildTime,
		environment: environment,
		startTime:   time.Now(),
	}
}

// Live implements GET /health/live
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   h.version,
	})
}

// Ready implements GET /health/ready
func (h *HealthHandler) Ready(c *gin.Context) {
	resp, ok := h.healthSvc.Ready(c.Request.Context())
	if !ok {
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Startup implements GET /health/startup
func (h *HealthHandler) Startup(c *gin.Context) {
	ok := h.healthSvc.Startup()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   h.version,
			"message":   "Application startup is in progress or failed",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   h.version,
		"message":   "Application startup completed successfully",
	})
}

// Check checks liveness (for backward compatibility).
func (h *HealthHandler) Check(c *gin.Context) {
	h.Live(c)
}

// CheckReady checks readiness (for backward compatibility).
func (h *HealthHandler) CheckReady(c *gin.Context) {
	h.Ready(c)
}
