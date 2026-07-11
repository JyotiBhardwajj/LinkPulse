// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"

	"linkpulse/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnalyticsRepository defines database operations for link click analytics.
type AnalyticsRepository interface {
	Create(ctx context.Context, click *models.Analytics) error
	GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error)

	// Placeholder methods for future reporting/dashboard metrics.
	// These complex queries will bypass GORM's standard ORM generation and use Raw SQL.
	GetBrowserDistribution(ctx context.Context, linkID uuid.UUID) (map[string]int64, error)
	GetClicksOverTime(ctx context.Context, linkID uuid.UUID, interval string) ([]models.ClickTimeMetric, error)
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

/*
================================================================================
SENIOR ENGINEERING RATIONALE: RAW SQL FOR AGGREGATION & REPORTING
================================================================================
For high-traffic aggregation and analytics dashboard queries, raw SQL is strongly
preferred over GORM's built-in query builder for the following reasons:

1. QUERY OPTIMIZATION: Aggregation queries require specific database-level behaviors
   (such as grouping, date truncations, custom interval buckets, and window functions).
   GORM's abstraction layers make it difficult to write fine-grained PostgreSQL optimizations.
2. LATENCY CONTROL: Raw SQL allows the engine to utilize custom composite indexes
   (e.g., idx_analytics_link_clicked) directly, ensuring sub-millisecond execution.
3. PREDICTABLE SQL: GORM can generate complex, nested subqueries and joins
   automatically, which can cause unexpected full-table scans. Raw SQL keeps execution paths
   explicit and audit-friendly.
================================================================================
*/

// GetBrowserDistribution aggregates click counts grouped by browser using GORM's Raw SQL interface.
func (r *analyticsRepository) GetBrowserDistribution(ctx context.Context, linkID uuid.UUID) (map[string]int64, error) {
	// PLACEHOLDER: Future implementation details will map raw SQL:
	// "SELECT browser, COUNT(*) FROM analytics WHERE link_id = ? GROUP BY browser"
	// utilizing r.db.WithContext(ctx).Raw(query, linkID).Scan(...)
	return make(map[string]int64), nil
}

// GetClicksOverTime aggregates click events by timeframe interval (e.g. 'day', 'hour') using GORM's Raw SQL interface.
func (r *analyticsRepository) GetClicksOverTime(ctx context.Context, linkID uuid.UUID, interval string) ([]models.ClickTimeMetric, error) {
	// PLACEHOLDER: Future implementation details will map raw SQL:
	// "SELECT date_trunc(?, clicked_at) AS time_bucket, COUNT(*) FROM analytics WHERE link_id = ? GROUP BY time_bucket ORDER BY time_bucket ASC"
	// utilizing date truncation functions native to PostgreSQL.
	return []models.ClickTimeMetric{}, nil
}
