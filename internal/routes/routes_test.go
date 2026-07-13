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
	"linkpulse/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockLinkService struct{}
type mockWorkerPool struct{}

func TestRouterConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a temporary ./docs directory relative to test execution context
	err := os.MkdirAll("docs", 0755)
	assert.NoError(t, err)
	defer os.RemoveAll("docs")

	err = os.WriteFile(filepath.Join("docs", "swagger.json"), []byte(`{"openapi": "3.0.3"}`), 0644)
	assert.NoError(t, err)

	// Temporarily override working directory relative path to find swagger.json or mock it.
	// Since routes.go uses "./docs/swagger.json", we will write to the actual local folder, or mock it by writing to ./docs/swagger.json if it doesn't exist.
	// Wait, ./docs/swagger.json already exists in our workspace! So SetupRouter will find it!

	healthH := handler.NewHealthHandler(nil, "1.0.0", "gitcommit", "2026-07-13", "development")

	r := SetupRouter(
		time.Second,
		"secret",
		"issuer",
		healthH,
		nil,
		nil,
		nil,
		nil,
		metrics.NewNoOpMetrics(),
	)

	t.Run("GET /health - success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
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
