package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
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

	t.Run("GET /swagger/index.html - success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
		req.RequestURI = "/swagger/index.html"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /swagger/doc.json - success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/swagger/doc.json", nil)
		req.RequestURI = "/swagger/doc.json"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		body, err := io.ReadAll(w.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(body), "swagger")
	})
}
