// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"
	"fmt"
	"time"

	"linkpulse/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnalyticsRepository defines database operations for link click analytics.
type AnalyticsRepository interface {
	Create(ctx context.Context, click *models.Analytics) error
	GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error)
	GetOverview(ctx context.Context, userID uuid.UUID) (*models.AnalyticsOverview, error)
	GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeMetric, error)
	GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error)
	GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error)
	GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error)
	GetTopLinks(ctx context.Context, userID uuid.UUID, limit int) ([]models.TopLinkMetric, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

// NewAnalyticsRepository creates a new AnalyticsRepository.
func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// Create inserts a click analytics record (Standard CRUD - GORM is ideal here).
func (r *analyticsRepository) Create(ctx context.Context, click *models.Analytics) error {
	return r.db.WithContext(ctx).Create(click).Error
}

// GetClicksCount returns the total number of click events for a specific link ID.
func (r *analyticsRepository) GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Analytics{}).
		Where("link_id = ?", linkID).
		Count(&count).Error
	return count, err
}

// GetOverview aggregates system-wide and timeline metrics for a user's link portfolio.
func (r *analyticsRepository) GetOverview(ctx context.Context, userID uuid.UUID) (*models.AnalyticsOverview, error) {
	var overview models.AnalyticsOverview
	now := time.Now()

	// 1. Links statistics (scoped to UserID using GORM indexing)
	err := r.db.WithContext(ctx).Model(&models.Link{}).
		Where("user_id = ?", userID).
		Count(&overview.TotalLinks).Error
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Model(&models.Link{}).
		Where("user_id = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)", userID, true, now).
		Count(&overview.ActiveLinks).Error
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Model(&models.Link{}).
		Where("user_id = ? AND (is_active = ? OR (expires_at IS NOT NULL AND expires_at <= ?))", userID, false, now).
		Count(&overview.InactiveLinks).Error
	if err != nil {
		return nil, err
	}

	// 2. Click counts aggregations (raw SQL joins to target idx_analytics_link_clicked)
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(a.id)
		FROM analytics a
		JOIN links l ON a.link_id = l.id
		WHERE l.user_id = ? AND l.deleted_at IS NULL`, userID).
		Scan(&overview.TotalClicks).Error
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(a.id)
		FROM analytics a
		JOIN links l ON a.link_id = l.id
		WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= CURRENT_DATE`, userID).
		Scan(&overview.TodayClicks).Error
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(a.id)
		FROM analytics a
		JOIN links l ON a.link_id = l.id
		WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= ?`, userID, now.Add(-7*24*time.Hour)).
		Scan(&overview.Last7DaysClicks).Error
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(a.id)
		FROM analytics a
		JOIN links l ON a.link_id = l.id
		WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= ?`, userID, now.Add(-30*24*time.Hour)).
		Scan(&overview.Last30DaysClicks).Error
	if err != nil {
		return nil, err
	}

	return &overview, nil
}

// GetClicksOverTime aggregates click counts grouped by interval using GORM's Raw SQL interface.
func (r *analyticsRepository) GetClicksOverTime(ctx context.Context, q models.AnalyticsQuery) ([]models.ClickTimeMetric, error) {
	intervalFunc := "day"
	switch q.Interval {
	case "hour", "day", "week", "month":
		intervalFunc = q.Interval
	default:
		return nil, fmt.Errorf("unwhitelisted time interval: %s", q.Interval)
	}

	var results []models.ClickTimeMetric
	var query string
	var args []interface{}

	if q.LinkID != nil {
		query = fmt.Sprintf(`
			SELECT date_trunc('%s', clicked_at) AS time_bucket, COUNT(*) AS click_count
			FROM analytics
			WHERE link_id = ? AND clicked_at >= ? AND clicked_at <= ?
			GROUP BY time_bucket
			ORDER BY time_bucket ASC`, intervalFunc)
		args = []interface{}{*q.LinkID, q.StartDate, q.EndDate}
	} else {
		query = fmt.Sprintf(`
			SELECT date_trunc('%s', a.clicked_at) AS time_bucket, COUNT(*) AS click_count
			FROM analytics a
			JOIN links l ON a.link_id = l.id
			WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= ? AND a.clicked_at <= ?
			GROUP BY time_bucket
			ORDER BY time_bucket ASC`, intervalFunc)
		args = []interface{}{q.UserID, q.StartDate, q.EndDate}
	}

	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	return results, err
}

// GetBrowserDistribution aggregates click counts grouped by browser using GORM's Raw SQL interface.
func (r *analyticsRepository) GetBrowserDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	var rows []struct {
		Browser string
		Count   int64
	}
	var query string
	var args []interface{}

	if q.LinkID != nil {
		query = `
			SELECT COALESCE(browser, 'Unknown') AS browser, COUNT(*) AS count
			FROM analytics
			WHERE link_id = ? AND clicked_at >= ? AND clicked_at <= ?
			GROUP BY browser`
		args = []interface{}{*q.LinkID, q.StartDate, q.EndDate}
	} else {
		query = `
			SELECT COALESCE(a.browser, 'Unknown') AS browser, COUNT(*) AS count
			FROM analytics a
			JOIN links l ON a.link_id = l.id
			WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= ? AND a.clicked_at <= ?
			GROUP BY a.browser`
		args = []interface{}{q.UserID, q.StartDate, q.EndDate}
	}

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	dist := make(map[string]int64)
	for _, r := range rows {
		dist[r.Browser] = r.Count
	}
	return dist, nil
}

// GetDeviceDistribution aggregates click counts grouped by device using GORM's Raw SQL interface.
func (r *analyticsRepository) GetDeviceDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	var rows []struct {
		Device string
		Count  int64
	}
	var query string
	var args []interface{}

	if q.LinkID != nil {
		query = `
			SELECT COALESCE(device, 'Unknown') AS device, COUNT(*) AS count
			FROM analytics
			WHERE link_id = ? AND clicked_at >= ? AND clicked_at <= ?
			GROUP BY device`
		args = []interface{}{*q.LinkID, q.StartDate, q.EndDate}
	} else {
		query = `
			SELECT COALESCE(a.device, 'Unknown') AS device, COUNT(*) AS count
			FROM analytics a
			JOIN links l ON a.link_id = l.id
			WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= ? AND a.clicked_at <= ?
			GROUP BY a.device`
		args = []interface{}{q.UserID, q.StartDate, q.EndDate}
	}

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	dist := make(map[string]int64)
	for _, r := range rows {
		dist[r.Device] = r.Count
	}
	return dist, nil
}

// GetReferrerDistribution aggregates referring domains grouped by domains.
func (r *analyticsRepository) GetReferrerDistribution(ctx context.Context, q models.AnalyticsQuery) (map[string]int64, error) {
	var rows []struct {
		Domain string
		Count  int64
	}
	var query string
	var args []interface{}

	if q.LinkID != nil {
		query = `
			SELECT COALESCE(NULLIF(regexp_replace(referrer, '^https?://([^/]+).*$', '\1'), ''), 'Direct/Unknown') AS domain, COUNT(*) AS count
			FROM analytics
			WHERE link_id = ? AND clicked_at >= ? AND clicked_at <= ?
			GROUP BY domain
			ORDER BY count DESC
			LIMIT ?`
		args = []interface{}{*q.LinkID, q.StartDate, q.EndDate, q.Limit}
	} else {
		query = `
			SELECT COALESCE(NULLIF(regexp_replace(a.referrer, '^https?://([^/]+).*$', '\1'), ''), 'Direct/Unknown') AS domain, COUNT(*) AS count
			FROM analytics a
			JOIN links l ON a.link_id = l.id
			WHERE l.user_id = ? AND l.deleted_at IS NULL AND a.clicked_at >= ? AND a.clicked_at <= ?
			GROUP BY domain
			ORDER BY count DESC
			LIMIT ?`
		args = []interface{}{q.UserID, q.StartDate, q.EndDate, q.Limit}
	}

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	dist := make(map[string]int64)
	for _, r := range rows {
		dist[r.Domain] = r.Count
	}
	return dist, nil
}

// GetTopLinks retrieves the top performing shortened links by click count.
func (r *analyticsRepository) GetTopLinks(ctx context.Context, userID uuid.UUID, limit int) ([]models.TopLinkMetric, error) {
	var results []models.TopLinkMetric
	query := `
		SELECT l.short_code, l.original_url, COUNT(a.id) AS click_count, MAX(a.clicked_at) AS last_clicked_at
		FROM links l
		LEFT JOIN analytics a ON l.id = a.link_id
		WHERE l.user_id = ? AND l.deleted_at IS NULL
		GROUP BY l.id, l.short_code, l.original_url
		ORDER BY click_count DESC, l.created_at DESC
		LIMIT ?`

	err := r.db.WithContext(ctx).Raw(query, userID, limit).Scan(&results).Error
	return results, err
}
