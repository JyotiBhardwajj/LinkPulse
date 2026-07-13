package app

import (
	"context"
	"testing"
	"time"

	"linkpulse/internal/metrics"
	"linkpulse/internal/models"
	"linkpulse/internal/worker"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockAnalyticsRepo struct{}

func (m *mockAnalyticsRepo) Create(ctx context.Context, click *models.Analytics) error {
	return nil
}

func (m *mockAnalyticsRepo) GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockAnalyticsRepo) GetOverview(ctx context.Context, userID uuid.UUID) (*models.AnalyticsOverview, error) {
	return nil, nil
}

func (m *mockAnalyticsRepo) GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeMetric, error) {
	return nil, nil
}

func (m *mockAnalyticsRepo) GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}

func (m *mockAnalyticsRepo) GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}

func (m *mockAnalyticsRepo) GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}

func (m *mockAnalyticsRepo) GetTopLinks(ctx context.Context, userID uuid.UUID, limit int) ([]models.TopLinkMetric, error) {
	return nil, nil
}

func TestWorkerPoolGracefulShutdown(t *testing.T) {
	m := metrics.NewNoOpMetrics()
	repo := &mockAnalyticsRepo{}

	// Create pool
	pool := worker.NewWorkerPool(repo, 2, 10, m)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit events
	err := pool.Submit(context.Background(), worker.ClickEvent{
		LinkID:    uuid.New(),
		Timestamp: time.Now(),
	})
	assert.NoError(t, err)

	// Cancel context to simulate cancellation during workerLoop execution
	cancel()

	// Shutdown should return quickly
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	shutdownCancel() // Cancel context immediately to force Canceled response

	err = pool.Shutdown(shutdownCtx)
	assert.ErrorIs(t, err, context.Canceled)
}
