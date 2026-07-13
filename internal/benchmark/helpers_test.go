package benchmark

import (
	"bytes"
	"context"
	"io"

	"linkpulse/internal/models"

	"github.com/google/uuid"
)

// noOpAnalyticsRepo is a no-op repository used in benchmarks to isolate worker
// queue and dispatch overhead from actual database I/O.
type noOpAnalyticsRepo struct{}

func (r *noOpAnalyticsRepo) Create(_ context.Context, _ *models.Analytics) error { return nil }

func (r *noOpAnalyticsRepo) GetClicksCount(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (r *noOpAnalyticsRepo) GetOverview(_ context.Context, _ uuid.UUID) (*models.AnalyticsOverview, error) {
	return nil, nil
}

func (r *noOpAnalyticsRepo) GetClicksOverTime(_ context.Context, _ models.AnalyticsQuery) ([]models.ClickTimeMetric, error) {
	return nil, nil
}

func (r *noOpAnalyticsRepo) GetBrowserDistribution(_ context.Context, _ models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}

func (r *noOpAnalyticsRepo) GetDeviceDistribution(_ context.Context, _ models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}

func (r *noOpAnalyticsRepo) GetReferrerDistribution(_ context.Context, _ models.AnalyticsQuery) (map[string]int64, error) {
	return nil, nil
}

func (r *noOpAnalyticsRepo) GetTopLinks(_ context.Context, _ uuid.UUID, _ int) ([]models.TopLinkMetric, error) {
	return nil, nil
}

// jsonReader returns an io.Reader wrapping a string for use in HTTP requests.
func jsonReader(s string) io.Reader {
	return bytes.NewBufferString(s)
}
