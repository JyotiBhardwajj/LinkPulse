package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkpulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReadinessService struct {
	resp *models.ReadyResponse
	err  error
}

func (m *mockReadinessService) Check(ctx context.Context) (*models.ReadyResponse, error) {
	return m.resp, m.err
}

func TestHealthHandler_Check(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockReadinessService{}
	h := NewHealthHandler(mockSvc, "1.2.3", "commit-hash", "build-time", "production")

	r := gin.New()
	r.GET("/health", h.Check)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                  `json:"success"`
		Message string                `json:"message"`
		Data    models.HealthResponse `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, "healthy", resp.Data.Status)
	assert.Equal(t, "1.2.3", resp.Data.Version)
	// Assert git commit is truncated to short SHA (first 7 characters)
	assert.Equal(t, "commit-", resp.Data.GitCommit)
}

func TestHealthHandler_CheckReady_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	readyResp := &models.ReadyResponse{
		Status:     "ready",
		Database:   "up",
		Redis:      "up",
		WorkerPool: "up",
		Timestamp:  time.Now().UTC(),
	}

	mockSvc := &mockReadinessService{
		resp: readyResp,
		err:  nil,
	}
	h := NewHealthHandler(mockSvc, "1.2.3", "commit-hash", "build-time", "production")

	r := gin.New()
	r.GET("/ready", h.CheckReady)

	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ReadyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "ready", resp.Status)
	assert.Equal(t, "up", resp.Database)
	assert.Equal(t, "up", resp.Redis)
	assert.Equal(t, "up", resp.WorkerPool)
}

func TestHealthHandler_CheckReady_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	readyResp := &models.ReadyResponse{
		Status:     "not_ready",
		Database:   "down",
		Redis:      "up",
		WorkerPool: "up",
		Timestamp:  time.Now().UTC(),
	}

	mockSvc := &mockReadinessService{
		resp: readyResp,
		err:  errors.New("postgres connection failed"),
	}
	h := NewHealthHandler(mockSvc, "1.2.3", "commit-hash", "build-time", "production")

	r := gin.New()
	r.GET("/ready", h.CheckReady)

	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp models.ReadyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "not_ready", resp.Status)
	assert.Equal(t, "down", resp.Database)
	assert.Equal(t, "up", resp.Redis)
}
