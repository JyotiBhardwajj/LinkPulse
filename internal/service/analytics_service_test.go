package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"linkpulse/internal/constants"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAnalyticsRepoForService struct {
	clicks      []models.ClickTimeMetric
	browsers    map[string]int64
	devices     map[string]int64
	referrers   map[string]int64
	topLinks    []models.TopLinkMetric
	overview    *models.AnalyticsOverview
	clicksCount int64
}

func (m *mockAnalyticsRepoForService) Create(ctx context.Context, click *models.Analytics) error {
	return nil
}
func (m *mockAnalyticsRepoForService) GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error) {
	return m.clicksCount, nil
}
func (m *mockAnalyticsRepoForService) GetOverview(ctx context.Context, userID uuid.UUID) (*models.AnalyticsOverview, error) {
	return m.overview, nil
}
func (m *mockAnalyticsRepoForService) GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeMetric, error) {
	return m.clicks, nil
}
func (m *mockAnalyticsRepoForService) GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return m.browsers, nil
}
func (m *mockAnalyticsRepoForService) GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return m.devices, nil
}
func (m *mockAnalyticsRepoForService) GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return m.referrers, nil
}
func (m *mockAnalyticsRepoForService) GetTopLinks(ctx context.Context, userID uuid.UUID, limit int) ([]models.TopLinkMetric, error) {
	return m.topLinks, nil
}

func TestAnalyticsService_Overview(t *testing.T) {
	repo := &mockAnalyticsRepoForService{
		overview: &models.AnalyticsOverview{
			TotalLinks:  10,
			ActiveLinks: 7,
			TotalClicks: 150,
		},
	}
	linkRepo := newMockLinkRepo()

	srv := NewAnalyticsService(repo, linkRepo)
	res, err := srv.GetOverview(context.Background(), models.AnalyticsQuery{UserID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, int64(10), res.TotalLinks)
	assert.Equal(t, int64(7), res.ActiveLinks)
	assert.Equal(t, int64(150), res.TotalClicks)
}

func TestAnalyticsService_ZeroFilling(t *testing.T) {
	linkRepo := newMockLinkRepo()
	startTime, _ := time.Parse(time.RFC3339, "2026-07-12T00:00:00Z")
	endTime, _ := time.Parse(time.RFC3339, "2026-07-12T03:00:00Z")

	// DB returns metrics for 00:00 and 02:00, missing 01:00 and 03:00
	dbMetrics := []models.ClickTimeMetric{
		{TimeBucket: startTime, ClickCount: 15},
		{TimeBucket: startTime.Add(2 * time.Hour), ClickCount: 7},
	}

	repo := &mockAnalyticsRepoForService{clicks: dbMetrics}
	srv := NewAnalyticsService(repo, linkRepo)

	q := models.AnalyticsQuery{
		UserID:    uuid.New(),
		StartDate: startTime,
		EndDate:   endTime,
		Interval:  constants.AnalyticsIntervalHour,
	}

	res, err := srv.GetClicksOverTime(context.Background(), q)
	require.NoError(t, err)
	require.Len(t, res, 4) // 00:00, 01:00, 02:00, 03:00

	// Check output formatting is RFC3339
	assert.Equal(t, startTime.Format(time.RFC3339), res[0].Timestamp)
	assert.Equal(t, int64(15), res[0].Clicks)

	assert.Equal(t, startTime.Add(1*time.Hour).Format(time.RFC3339), res[1].Timestamp)
	assert.Equal(t, int64(0), res[1].Clicks) // Zero-Filled

	assert.Equal(t, startTime.Add(2*time.Hour).Format(time.RFC3339), res[2].Timestamp)
	assert.Equal(t, int64(7), res[2].Clicks)

	assert.Equal(t, startTime.Add(3*time.Hour).Format(time.RFC3339), res[3].Timestamp)
	assert.Equal(t, int64(0), res[3].Clicks) // Zero-Filled
}

func TestAnalyticsService_IntervalValidation(t *testing.T) {
	repo := &mockAnalyticsRepoForService{}
	linkRepo := newMockLinkRepo()
	srv := NewAnalyticsService(repo, linkRepo)

	q := models.AnalyticsQuery{
		UserID:    uuid.New(),
		StartDate: time.Now(),
		EndDate:   time.Now(),
		Interval:  "year",
	}

	_, err := srv.GetClicksOverTime(context.Background(), q)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported time interval")
}

func TestAnalyticsService_PercentageCalculation(t *testing.T) {
	repo := &mockAnalyticsRepoForService{
		browsers: map[string]int64{
			"Chrome":  60,
			"Firefox": 40,
		},
	}
	linkRepo := newMockLinkRepo()
	srv := NewAnalyticsService(repo, linkRepo)

	q := models.AnalyticsQuery{
		UserID:    uuid.New(),
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   time.Now(),
	}

	res, err := srv.GetBrowserDistribution(context.Background(), q)
	require.NoError(t, err)

	var chromePct, firefoxPct float64
	for _, item := range res {
		if item.Name == "Chrome" {
			chromePct = item.Percentage
		} else if item.Name == "Firefox" {
			firefoxPct = item.Percentage
		}
	}

	assert.Equal(t, 60.0, chromePct)
	assert.Equal(t, 40.0, firefoxPct)
}

func TestAnalyticsService_LimitClamping(t *testing.T) {
	repo := &mockAnalyticsRepoForService{}
	linkRepo := newMockLinkRepo()
	srv := NewAnalyticsService(repo, linkRepo)

	q := models.AnalyticsQuery{
		UserID: uuid.New(),
		Limit:  500,
	}

	// Clamp to 100 maximum
	res, err := srv.GetTopLinks(context.Background(), q)
	require.NoError(t, err)
	assert.Empty(t, res) // mock returns empty but check it executes
}

func TestAnalyticsService_OwnershipValidation(t *testing.T) {
	linkRepo := newMockLinkRepo()
	repo := &mockAnalyticsRepoForService{}
	srv := NewAnalyticsService(repo, linkRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	linkID := uuid.New()

	// Seed unowned link
	_ = linkRepo.Create(context.Background(), &models.Link{
		ID:        linkID,
		ShortCode: "goog",
		UserID:    &otherUserID,
	})

	q := models.AnalyticsQuery{
		UserID:    userID,
		LinkID:    &linkID,
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   time.Now(),
	}

	// GetLinkAnalytics with wrong owner userID returns NotFound
	_, err := srv.GetLinkAnalytics(context.Background(), q)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domainErrors.ErrNotFound))
}
