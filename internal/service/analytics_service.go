package service

import (
	"context"
	"fmt"
	"time"

	"linkpulse/internal/constants"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/metrics"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"

	"github.com/google/uuid"
)

// AnalyticsService defines business logic for compiling and zero-filling aggregates.
type AnalyticsService interface {
	GetOverview(ctx context.Context, query models.AnalyticsQuery) (*models.AnalyticsOverview, error)
	GetClicksOverTime(ctx context.Context, query models.AnalyticsQuery) ([]models.ClickTimeSeriesPoint, error)
	GetTopLinks(ctx context.Context, query models.AnalyticsQuery) ([]models.TopLinkMetric, error)
	GetBrowserDistribution(ctx context.Context, query models.AnalyticsQuery) ([]models.DistributionItem, error)
	GetDeviceDistribution(ctx context.Context, query models.AnalyticsQuery) ([]models.DistributionItem, error)
	GetReferrerDistribution(ctx context.Context, query models.AnalyticsQuery) ([]models.DistributionItem, error)
	GetLinkAnalytics(ctx context.Context, query models.AnalyticsQuery) (*models.LinkAnalyticsResponse, error)
}

type analyticsService struct {
	analyticsRepo repository.AnalyticsRepository
	linkRepo      repository.LinkRepository
	metrics       metrics.Metrics
}

// NewAnalyticsService instantiates a new AnalyticsService implementation.
func NewAnalyticsService(analyticsRepo repository.AnalyticsRepository, linkRepo repository.LinkRepository, metricsTracker metrics.Metrics) AnalyticsService {
	return &analyticsService{
		analyticsRepo: analyticsRepo,
		linkRepo:      linkRepo,
		metrics:       metricsTracker,
	}
}

func (s *analyticsService) GetOverview(ctx context.Context, q models.AnalyticsQuery) (*models.AnalyticsOverview, error) {
	if q.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user ID", domainErrors.ErrInvalidInput)
	}
	return s.analyticsRepo.GetOverview(ctx, q.UserID)
}

func (s *analyticsService) GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeSeriesPoint, error) {
	// Standardize to UTC timezone
	q.StartDate = q.StartDate.UTC()
	q.EndDate = q.EndDate.UTC()

	if err := s.validateDateRange(q.StartDate, q.EndDate); err != nil {
		return nil, err
	}

	// 1. Whitelist Interval Protection using Constants
	switch q.Interval {
	case constants.AnalyticsIntervalHour, constants.AnalyticsIntervalDay, constants.AnalyticsIntervalWeek, constants.AnalyticsIntervalMonth:
		// Allowed whitelisted values
	default:
		return nil, fmt.Errorf("%w: unsupported time interval: %s", domainErrors.ErrInvalidInput, q.Interval)
	}

	dbMetrics, err := s.analyticsRepo.GetClicksOverTime(ctx, q)
	if err != nil {
		return nil, err
	}

	// 2. Zero-Fill time-series logic
	return s.zeroFillTimeSeries(dbMetrics, q.StartDate, q.EndDate, q.Interval), nil
}

func (s *analyticsService) GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) ([]models.DistributionItem, error) {
	q.StartDate = q.StartDate.UTC()
	q.EndDate = q.EndDate.UTC()

	if err := s.validateDateRange(q.StartDate, q.EndDate); err != nil {
		return nil, err
	}

	counts, err := s.analyticsRepo.GetBrowserDistribution(ctx, q)
	if err != nil {
		return nil, err
	}

	// Ensure whitelisted browser categories are present even with 0 counts
	browsers := []string{"Chrome", "Firefox", "Safari", "Edge", "Opera", "Unknown"}
	for _, b := range browsers {
		if _, ok := counts[b]; !ok {
			counts[b] = 0
		}
	}

	return s.calculatePercentages(counts), nil
}

func (s *analyticsService) GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) ([]models.DistributionItem, error) {
	q.StartDate = q.StartDate.UTC()
	q.EndDate = q.EndDate.UTC()

	if err := s.validateDateRange(q.StartDate, q.EndDate); err != nil {
		return nil, err
	}

	counts, err := s.analyticsRepo.GetDeviceDistribution(ctx, q)
	if err != nil {
		return nil, err
	}

	// Ensure categories are present
	devices := []string{"Desktop", "Mobile", "Tablet", "Unknown"}
	for _, d := range devices {
		if _, ok := counts[d]; !ok {
			counts[d] = 0
		}
	}

	return s.calculatePercentages(counts), nil
}

func (s *analyticsService) GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) ([]models.DistributionItem, error) {
	q.StartDate = q.StartDate.UTC()
	q.EndDate = q.EndDate.UTC()

	if err := s.validateDateRange(q.StartDate, q.EndDate); err != nil {
		return nil, err
	}

	// Clamp limits parameters
	q.Limit = s.clampLimit(q.Limit)

	counts, err := s.analyticsRepo.GetReferrerDistribution(ctx, q)
	if err != nil {
		return nil, err
	}

	return s.calculatePercentages(counts), nil
}

func (s *analyticsService) GetTopLinks(ctx context.Context, q models.AnalyticsQuery) ([]models.TopLinkMetric, error) {
	if q.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user ID", domainErrors.ErrInvalidInput)
	}

	q.Limit = s.clampLimit(q.Limit)
	return s.analyticsRepo.GetTopLinks(ctx, q.UserID, q.Limit)
}

func (s *analyticsService) GetLinkAnalytics(ctx context.Context, q models.AnalyticsQuery) (*models.LinkAnalyticsResponse, error) {
	if q.LinkID == nil {
		return nil, fmt.Errorf("%w: missing link ID", domainErrors.ErrInvalidInput)
	}

	q.StartDate = q.StartDate.UTC()
	q.EndDate = q.EndDate.UTC()

	if err := s.validateDateRange(q.StartDate, q.EndDate); err != nil {
		return nil, err
	}

	// Ownership Validation: Return 404 on unowned or missing ID for security prevention
	link, err := s.linkRepo.FindByID(ctx, *q.LinkID)
	if err != nil {
		return nil, domainErrors.ErrNotFound
	}
	if link.UserID == nil || *link.UserID != q.UserID {
		return nil, domainErrors.ErrNotFound
	}

	totalClicks, err := s.analyticsRepo.GetClicksCount(ctx, *q.LinkID)
	if err != nil {
		return nil, err
	}

	// Copy and set interval to Day for single link chart defaults
	chartQ := q
	chartQ.Interval = constants.AnalyticsIntervalDay
	clicksOverTime, err := s.GetClicksOverTime(ctx, chartQ)
	if err != nil {
		return nil, err
	}

	browsers, err := s.GetBrowserDistribution(ctx, q)
	if err != nil {
		return nil, err
	}

	devices, err := s.GetDeviceDistribution(ctx, q)
	if err != nil {
		return nil, err
	}

	referrersQ := q
	referrersQ.Limit = 10
	referrers, err := s.GetReferrerDistribution(ctx, referrersQ)
	if err != nil {
		return nil, err
	}

	return &models.LinkAnalyticsResponse{
		LinkID:              link.ID,
		OriginalURL:         link.OriginalURL,
		ShortCode:           link.ShortCode,
		TotalClicks:         totalClicks,
		ClicksOverTime:      clicksOverTime,
		BrowserDistribution: browsers,
		DeviceDistribution:  devices,
		TopReferrers:        referrers,
	}, nil
}

// Helpers
func (s *analyticsService) validateDateRange(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return fmt.Errorf("%w: start and end dates are required", domainErrors.ErrInvalidInput)
	}
	if start.After(end) {
		return fmt.Errorf("%w: start date must be before or equal to end date", domainErrors.ErrInvalidInput)
	}
	return nil
}

func (s *analyticsService) clampLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (s *analyticsService) calculatePercentages(counts map[string]int64) []models.DistributionItem {
	var total int64
	for _, c := range counts {
		total += c
	}

	var items []models.DistributionItem
	for name, count := range counts {
		var pct float64
		if total > 0 {
			pct = (float64(count) / float64(total)) * 100.0
		}
		items = append(items, models.DistributionItem{
			Name:       name,
			Count:      count,
			Percentage: pct,
		})
	}
	return items
}

func (s *analyticsService) zeroFillTimeSeries(metrics []models.ClickTimeMetric, start, end time.Time, interval string) []models.ClickTimeSeriesPoint {
	// 1. Index raw database points
	dbMap := make(map[string]int64)
	for _, m := range metrics {
		bucketKey := s.formatBucketKey(m.TimeBucket.UTC(), interval)
		dbMap[bucketKey] = m.ClickCount
	}

	// 2. Truncate start and end times to bucket alignments in UTC
	current := s.truncateToBucket(start.UTC(), interval)
	limit := s.truncateToBucket(end.UTC(), interval)

	var points []models.ClickTimeSeriesPoint

	// 3. Loop generating continuous interval buckets
	for !current.After(limit) {
		key := s.formatBucketKey(current, interval)
		clicks := dbMap[key] // defaults to 0 if not present

		points = append(points, models.ClickTimeSeriesPoint{
			Timestamp: key,
			Clicks:    clicks,
		})

		current = s.incrementBucket(current, interval)
	}

	return points
}

func (s *analyticsService) truncateToBucket(t time.Time, interval string) time.Time {
	switch interval {
	case constants.AnalyticsIntervalHour:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	case constants.AnalyticsIntervalDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	case constants.AnalyticsIntervalWeek:
		// Align to Monday
		for t.Weekday() != time.Monday {
			t = t.AddDate(0, 0, -1)
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	case constants.AnalyticsIntervalMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return t
	}
}

func (s *analyticsService) incrementBucket(t time.Time, interval string) time.Time {
	switch interval {
	case constants.AnalyticsIntervalHour:
		return t.Add(1 * time.Hour)
	case constants.AnalyticsIntervalDay:
		return t.AddDate(0, 0, 1)
	case constants.AnalyticsIntervalWeek:
		return t.AddDate(0, 0, 7)
	case constants.AnalyticsIntervalMonth:
		return t.AddDate(0, 1, 0)
	default:
		return t
	}
}

func (s *analyticsService) formatBucketKey(t time.Time, interval string) string {
	// Return timestamps in RFC3339 format to ensure standardization
	return t.Format(time.RFC3339)
}
