// Package handler implements HTTP controllers and request parsers.
package handler

import (
	"context"
	"net/http"
	"time"

	"linkpulse/internal/cache"
	"linkpulse/internal/database"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// HealthHandler manages application diagnostic checks.
type HealthHandler struct {
	db          *database.PostgresDB
	redis       *cache.RedisClient
	version     string
	gitCommit   string
	environment string
	startTime   time.Time
}

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler(db *database.PostgresDB, redis *cache.RedisClient, version, gitCommit, environment string) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redis:       redis,
		version:     version,
		gitCommit:   gitCommit,
		environment: environment,
		startTime:   time.Now(),
	}
}

type checkDetail struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
}

type healthResponse struct {
	Status        string      `json:"status"`
	Postgres      checkDetail `json:"postgres"`
	Redis         checkDetail `json:"redis"`
	Version       string      `json:"version"`
	GitCommit     string      `json:"git_commit"`
	Environment   string      `json:"environment"`
	UptimeSeconds int64       `json:"uptime_seconds"`
	Timestamp     string      `json:"timestamp"`
}

// Check verifies connection status and latencies for dependent services.
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	overallStatus := "healthy"

	// Postgres Health check
	pgStart := time.Now()
	pgStatus := "up"
	var pgLatency int64
	if err := h.db.Ping(); err != nil {
		pgStatus = "down"
		overallStatus = "unhealthy"
	} else {
		pgLatency = time.Since(pgStart).Milliseconds()
	}

	// Redis Health check
	rStart := time.Now()
	rStatus := "up"
	var rLatency int64
	if err := h.redis.Ping(ctx); err != nil {
		rStatus = "down"
		overallStatus = "unhealthy"
	} else {
		rLatency = time.Since(rStart).Milliseconds()
	}

	uptime := int64(time.Since(h.startTime).Seconds())

	resp := healthResponse{
		Status: overallStatus,
		Postgres: checkDetail{
			Status:    pgStatus,
			LatencyMS: pgLatency,
		},
		Redis: checkDetail{
			Status:    rStatus,
			LatencyMS: rLatency,
		},
		Version:       h.version,
		GitCommit:     h.gitCommit,
		Environment:   h.environment,
		UptimeSeconds: uptime,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	utils.SendSuccess(c, http.StatusOK, "Health check completed", resp)
}
