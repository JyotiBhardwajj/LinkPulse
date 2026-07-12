package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkpulse/internal/auth"
	"linkpulse/internal/middleware"
	"linkpulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAnalyticsService struct {
	overview   *models.AnalyticsOverview
	timeSeries []models.ClickTimeSeriesPoint
	devices    []models.DistributionItem
	browsers   []models.DistributionItem
	referrers  []models.DistributionItem
	topLinks   []models.TopLinkMetric
	report     *models.LinkAnalyticsResponse
}

func (m *mockAnalyticsService) GetOverview(ctx context.Context, q models.AnalyticsQuery) (*models.AnalyticsOverview, error) {
	return m.overview, nil
}
func (m *mockAnalyticsService) GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeSeriesPoint, error) {
	return m.timeSeries, nil
}
func (m *mockAnalyticsService) GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) ([]models.DistributionItem, error) {
	return m.browsers, nil
}
func (m *mockAnalyticsService) GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) ([]models.DistributionItem, error) {
	return m.devices, nil
}
func (m *mockAnalyticsService) GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) ([]models.DistributionItem, error) {
	return m.referrers, nil
}
func (m *mockAnalyticsService) GetTopLinks(ctx context.Context, q models.AnalyticsQuery) ([]models.TopLinkMetric, error) {
	return m.topLinks, nil
}
func (m *mockAnalyticsService) GetLinkAnalytics(ctx context.Context, q models.AnalyticsQuery) (*models.LinkAnalyticsResponse, error) {
	return m.report, nil
}

func TestAnalyticsHandler_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSrv := &mockAnalyticsService{
		overview: &models.AnalyticsOverview{
			TotalLinks:  5,
			TotalClicks: 42,
		},
		timeSeries: []models.ClickTimeSeriesPoint{
			{Timestamp: "2026-07-12T00:00:00Z", Clicks: 42},
		},
	}
	h := NewAnalyticsHandler(mockSrv)

	secret := "handlertestsecretkeythatisreallylong"
	issuer := "linkpulse-api"
	accessTTL := 5 * time.Minute
	authMiddleware := middleware.Auth(secret, issuer)

	r := gin.New()
	api := r.Group("/api/v1", authMiddleware)
	{
		analytics := api.Group("/analytics")
		{
			analytics.GET("/overview", h.GetOverview)
			analytics.GET("/clicks", h.GetClicksOverTime)
		}
	}

	claimsUserID := uuid.New()
	token, _ := auth.GenerateAccessToken(claimsUserID, "user@example.com", secret, accessTTL, issuer)

	t.Run("GET /analytics/overview returns aggregated overview stats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/analytics/overview", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool                     `json:"success"`
			Data    models.AnalyticsOverview `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, int64(5), resp.Data.TotalLinks)
		assert.Equal(t, int64(42), resp.Data.TotalClicks)
	})

	t.Run("GET /analytics/clicks returns time series data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/analytics/clicks?interval=day", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool                          `json:"success"`
			Data    []models.ClickTimeSeriesPoint `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		require.Len(t, resp.Data, 1)
		assert.Equal(t, "2026-07-12T00:00:00Z", resp.Data[0].Timestamp)
		assert.Equal(t, int64(42), resp.Data[0].Clicks)
	})
}
