package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"linkpulse/internal/health"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHealthService struct {
	live    bool
	ready   health.HealthResponse
	readyOk bool
	startup bool
}

func (m *mockHealthService) Register(checker health.Checker) {}
func (m *mockHealthService) CheckAll(ctx context.Context) health.HealthResponse {
	return m.ready
}
func (m *mockHealthService) Live() bool {
	return m.live
}
func (m *mockHealthService) Ready(ctx context.Context) (health.HealthResponse, bool) {
	return m.ready, m.readyOk
}
func (m *mockHealthService) Startup() bool {
	return m.startup
}
func (m *mockHealthService) SetStartupComplete() {}

func TestHealthHandler_Live(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockHealthService{live: true}
	h := NewHealthHandler(mockSvc, "1.2.3", "commit-hash", "build-time", "production")

	r := gin.New()
	r.GET("/health/live", h.Live)

	req, _ := http.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
		Version   string `json:"version"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "1.2.3", resp.Version)
}

func TestHealthHandler_Ready_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	readyResp := health.HealthResponse{
		Status: "healthy",
		Checks: []health.DependencyResult{
			{Name: "postgres", Status: "healthy", Critical: true},
			{Name: "redis", Status: "healthy", Critical: false},
		},
	}

	mockSvc := &mockHealthService{
		ready:   readyResp,
		readyOk: true,
	}
	h := NewHealthHandler(mockSvc, "1.2.3", "commit-hash", "build-time", "production")

	r := gin.New()
	r.GET("/health/ready", h.Ready)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp health.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, 2, len(resp.Checks))
}

func TestHealthHandler_Ready_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	unhealthyResp := health.HealthResponse{
		Status: "unhealthy",
		Checks: []health.DependencyResult{
			{Name: "postgres", Status: "unhealthy", Critical: true, Error: "db failed"},
		},
	}

	mockSvc := &mockHealthService{
		ready:   unhealthyResp,
		readyOk: false,
	}
	h := NewHealthHandler(mockSvc, "1.2.3", "commit-hash", "build-time", "production")

	r := gin.New()
	r.GET("/health/ready", h.Ready)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp health.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "unhealthy", resp.Status)
}
