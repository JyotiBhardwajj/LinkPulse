package handler

import (
	"net/http"
	"time"

	"linkpulse/internal/models"
	"linkpulse/internal/service"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// HealthHandler manages application diagnostic checks.
type HealthHandler struct {
	readyService service.ReadinessService
	version      string
	gitCommit    string
	buildTime    string
	environment  string
	startTime    time.Time
}

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler(readyService service.ReadinessService, version, gitCommit, buildTime, environment string) *HealthHandler {
	return &HealthHandler{
		readyService: readyService,
		version:      version,
		gitCommit:    gitCommit,
		buildTime:    buildTime,
		environment:  environment,
		startTime:    time.Now(),
	}
}

// Check verifies the application liveness status without dependency checking.
func (h *HealthHandler) Check(c *gin.Context) {
	commit := h.gitCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}

	resp := models.HealthResponse{
		Status:    "healthy",
		Version:   h.version,
		GitCommit: commit,
		Timestamp: time.Now().UTC(),
	}

	utils.SendSuccess(c, http.StatusOK, "Liveness check completed", resp)
}

// CheckReady validates all backend dependencies (readiness).
func (h *HealthHandler) CheckReady(c *gin.Context) {
	resp, err := h.readyService.Check(c.Request.Context())
	if err != nil {
		// Respond with 503 if database, cache, or worker pool is down
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}
