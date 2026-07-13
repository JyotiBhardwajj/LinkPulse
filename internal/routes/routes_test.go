package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"linkpulse/internal/handler"
	"linkpulse/internal/health"
	"linkpulse/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouterConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a temporary ./docs directory relative to test execution context
	err := os.MkdirAll("docs", 0755)
	assert.NoError(t, err)
	defer os.RemoveAll("docs")

	err = os.WriteFile(filepath.Join("docs", "swagger.json"), []byte(`{"openapi": "3.0.3"}`), 0644)
	assert.NoError(t, err)

	m := metrics.NewNoOpMetrics()
	rs := health.NewReadinessState(m)
	hs := health.NewHealthService(rs, "1.0.0", 50*time.Millisecond, m)
	healthH := handler.NewHealthHandler(hs, "1.0.0", "gitcommit", "2026-07-13", "development")

	r := SetupRouter(
		time.Second,
		"secret",
		"issuer",
		healthH,
		nil,
		nil,
		nil,
		nil,
		m,
	)

	t.Run("GET /health/live - success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/live", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
	})

	t.Run("GET /health/ready - 503 before ready set", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/ready", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("GET /health/ready - 200 after ready set", func(t *testing.T) {
		rs.SetReady()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/ready", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /health/startup - 503 before startupComplete", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/startup", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("GET /health/startup - 200 after startupComplete", func(t *testing.T) {
		hs.SetStartupComplete()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/startup", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /docs/swagger.json - success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/docs/swagger.json", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		body, err := io.ReadAll(w.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(body), "openapi")
	})
}
