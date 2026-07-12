package database

import (
	"sync"
	"time"

	"linkpulse/internal/metrics"

	"gorm.io/gorm"
)

const (
	metricsStartTimeKey = "metrics:start_time"
)

// MetricsPlugin intercepts GORM DB operations to measure query execution duration.
type MetricsPlugin struct {
	tracker metrics.Metrics
	once    sync.Once
}

// NewMetricsPlugin creates a new instance of GORM metrics plugin.
func NewMetricsPlugin(tracker metrics.Metrics) *MetricsPlugin {
	return &MetricsPlugin{tracker: tracker}
}

// Name returns the unique plugin name.
func (p *MetricsPlugin) Name() string {
	return "linkpulse:metrics"
}

// Initialize registers GORM callbacks for observability metrics.
func (p *MetricsPlugin) Initialize(db *gorm.DB) error {
	var err error
	p.once.Do(func() {
		// Register 'Before' hooks to record query start times
		_ = db.Callback().Create().Before("gorm:create").Register("metrics:before_create", p.before)
		_ = db.Callback().Query().Before("gorm:query").Register("metrics:before_query", p.before)
		_ = db.Callback().Update().Before("gorm:update").Register("metrics:before_update", p.before)
		_ = db.Callback().Delete().Before("gorm:delete").Register("metrics:before_delete", p.before)
		_ = db.Callback().Row().Before("gorm:row").Register("metrics:before_row", p.before)

		// Register 'After' hooks to log durations
		_ = db.Callback().Create().After("gorm:create").Register("metrics:after_create", func(d *gorm.DB) { p.after(d, "create") })
		_ = db.Callback().Query().After("gorm:query").Register("metrics:after_query", func(d *gorm.DB) { p.after(d, "query") })
		_ = db.Callback().Update().After("gorm:update").Register("metrics:after_update", func(d *gorm.DB) { p.after(d, "update") })
		_ = db.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", func(d *gorm.DB) { p.after(d, "delete") })
		_ = db.Callback().Row().After("gorm:row").Register("metrics:after_row", func(d *gorm.DB) { p.after(d, "row") })
	})
	return err
}

func (p *MetricsPlugin) before(db *gorm.DB) {
	if db != nil {
		db.InstanceSet(metricsStartTimeKey, time.Now())
	}
}

func (p *MetricsPlugin) after(db *gorm.DB, operation string) {
	if db == nil {
		return
	}
	if val, ok := db.InstanceGet(metricsStartTimeKey); ok {
		if startTime, ok := val.(time.Time); ok {
			duration := time.Since(startTime)
			repo := "unknown"
			if db.Statement != nil && db.Statement.Table != "" {
				repo = db.Statement.Table
			} else if db.Statement != nil && db.Statement.Schema != nil {
				repo = db.Statement.Schema.Table
			}
			p.tracker.RecordDBQuery(repo, operation, duration)
		}
	}
}
