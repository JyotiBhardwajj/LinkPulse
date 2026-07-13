// Package metrics provides observability and monitoring backed by Prometheus.
package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusMetrics struct {
	registry *prometheus.Registry

	// HTTP metrics
	httpRequests   *prometheus.CounterVec
	httpRequestDur *prometheus.HistogramVec

	// Cache metrics
	cacheHits   prometheus.Counter
	cacheMisses prometheus.Counter
	cacheErrors prometheus.Counter

	// Worker metrics
	workerProcessed prometheus.Counter
	workerDropped   prometheus.Counter
	workerQueueSize prometheus.Gauge
	workerActive    prometheus.Gauge

	// Link metrics
	linksCreated  prometheus.Counter
	linksUpdated  prometheus.Counter
	linksDeleted  prometheus.Counter
	linksResolved prometheus.Counter

	// Auth metrics
	loginSuccess   prometheus.Counter
	loginFailure   prometheus.Counter
	refreshSuccess prometheus.Counter
	refreshFailure prometheus.Counter
	logout         prometheus.Counter

	// Analytics metrics
	analyticsWrites prometheus.Counter
	analyticsErrors prometheus.Counter

	// DB metrics
	dbQueryDur *prometheus.HistogramVec

	// Day 10 health metrics
	healthCheckDur prometheus.Histogram
	readinessState prometheus.Gauge
	startupDur     prometheus.Gauge
}

var (
	prodMetrics  Metrics
	prodRegistry *prometheus.Registry
	prodOnce     sync.Once
)

// GetProductionMetrics retrieves or instantiates the singleton production Prometheus metrics tracker.
func GetProductionMetrics(namespace, subsystem string) (Metrics, *prometheus.Registry) {
	prodOnce.Do(func() {
		prodMetrics, prodRegistry = NewPrometheusMetrics(namespace, subsystem)
	})
	return prodMetrics, prodRegistry
}

// NewPrometheusMetrics instantiates a new Metrics implementation backed by a dedicated, isolated prometheus.Registry.
func NewPrometheusMetrics(namespace, subsystem string) (Metrics, *prometheus.Registry) {
	reg := prometheus.NewRegistry()

	pm := &prometheusMetrics{
		registry: reg,

		httpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests processed.",
			},
			[]string{"method", "route", "status"},
		),

		httpRequestDur: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "Latency of HTTP requests in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "route"},
		),

		cacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "redis_cache_hits_total",
				Help:      "Total number of Redis cache hits.",
			},
		),

		cacheMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "redis_cache_misses_total",
				Help:      "Total number of Redis cache misses.",
			},
		),

		cacheErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "redis_cache_errors_total",
				Help:      "Total number of Redis cache errors.",
			},
		),

		workerProcessed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_events_processed_total",
				Help:      "Total number of click events processed by background workers.",
			},
		),

		workerDropped: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_events_dropped_total",
				Help:      "Total number of click events dropped by background workers due to queue overflow.",
			},
		),

		workerQueueSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_queue_size",
				Help:      "Current number of pending click events in the worker queue.",
			},
		),

		workerActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "worker_active_workers",
				Help:      "Number of currently active worker routines.",
			},
		),

		linksCreated: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "links_created_total",
				Help:      "Total count of shortened links created.",
			},
		),

		linksUpdated: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "links_updated_total",
				Help:      "Total count of shortened links updated.",
			},
		),

		linksDeleted: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "links_deleted_total",
				Help:      "Total count of shortened links deleted.",
			},
		),

		linksResolved: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "links_resolved_total",
				Help:      "Total count of shortened links resolved / redirected.",
			},
		),

		loginSuccess: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "login_success_total",
				Help:      "Total count of successful login attempts.",
			},
		),

		loginFailure: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "login_failure_total",
				Help:      "Total count of failed login attempts.",
			},
		),

		refreshSuccess: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "refresh_success_total",
				Help:      "Total count of successful refresh token cycles.",
			},
		),

		refreshFailure: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "refresh_failure_total",
				Help:      "Total count of failed refresh token cycles.",
			},
		),

		logout: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "logout_total",
				Help:      "Total count of user logout events.",
			},
		),

		analyticsWrites: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "analytics_events_written_total",
				Help:      "Total number of click analytics events persisted to the database.",
			},
		),

		analyticsErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "analytics_processing_errors_total",
				Help:      "Total number of background worker analytics persistence errors.",
			},
		),

		dbQueryDur: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_query_duration_seconds",
				Help:      "Database query execution latencies in seconds.",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
			},
			[]string{"repository", "operation"},
		),

		healthCheckDur: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "health_check_duration_seconds",
				Help:      "Duration of dependency health checks in seconds.",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
		),

		readinessState: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "readiness_state",
				Help:      "Current readiness status of the application (1 for ready, 0 for not ready).",
			},
		),

		startupDur: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "startup_duration_seconds",
				Help:      "Application boot/startup completion latency in seconds.",
			},
		),
	}

	// Register collectors on the dedicated registry instance
	reg.MustRegister(
		pm.httpRequests,
		pm.httpRequestDur,
		pm.cacheHits,
		pm.cacheMisses,
		pm.cacheErrors,
		pm.workerProcessed,
		pm.workerDropped,
		pm.workerQueueSize,
		pm.workerActive,
		pm.linksCreated,
		pm.linksUpdated,
		pm.linksDeleted,
		pm.linksResolved,
		pm.loginSuccess,
		pm.loginFailure,
		pm.refreshSuccess,
		pm.refreshFailure,
		pm.logout,
		pm.analyticsWrites,
		pm.analyticsErrors,
		pm.dbQueryDur,
		pm.healthCheckDur,
		pm.readinessState,
		pm.startupDur,
	)

	return pm, reg
}

func (p *prometheusMetrics) RecordHTTPRequest(method, route, status string) {
	p.httpRequests.WithLabelValues(method, route, status).Inc()
}

func (p *prometheusMetrics) RecordRequestDuration(method, route string, duration time.Duration) {
	p.httpRequestDur.WithLabelValues(method, route).Observe(duration.Seconds())
}

func (p *prometheusMetrics) RecordCacheHit() {
	p.cacheHits.Inc()
}

func (p *prometheusMetrics) RecordCacheMiss() {
	p.cacheMisses.Inc()
}

func (p *prometheusMetrics) RecordCacheError() {
	p.cacheErrors.Inc()
}

func (p *prometheusMetrics) RecordWorkerProcessed() {
	p.workerProcessed.Inc()
}

func (p *prometheusMetrics) RecordWorkerDropped() {
	p.workerDropped.Inc()
}

func (p *prometheusMetrics) RecordWorkerQueueSize(size int) {
	p.workerQueueSize.Set(float64(size))
}

func (p *prometheusMetrics) RecordWorkerActive(count int) {
	p.workerActive.Set(float64(count))
}

func (p *prometheusMetrics) RecordLoginSuccess() {
	p.loginSuccess.Inc()
}

func (p *prometheusMetrics) RecordLoginFailure() {
	p.loginFailure.Inc()
}

func (p *prometheusMetrics) RecordRefreshSuccess() {
	p.refreshSuccess.Inc()
}

func (p *prometheusMetrics) RecordRefreshFailure() {
	p.refreshFailure.Inc()
}

func (p *prometheusMetrics) RecordLogout() {
	p.logout.Inc()
}

func (p *prometheusMetrics) RecordDBQuery(repository, operation string, duration time.Duration) {
	p.dbQueryDur.WithLabelValues(repository, operation).Observe(duration.Seconds())
}

func (p *prometheusMetrics) RecordLinkCreated() {
	p.linksCreated.Inc()
}

func (p *prometheusMetrics) RecordLinkUpdated() {
	p.linksUpdated.Inc()
}

func (p *prometheusMetrics) RecordLinkDeleted() {
	p.linksDeleted.Inc()
}

func (p *prometheusMetrics) RecordLinkResolved() {
	p.linksResolved.Inc()
}

func (p *prometheusMetrics) RecordAnalyticsWrite() {
	p.analyticsWrites.Inc()
}

func (p *prometheusMetrics) RecordAnalyticsError() {
	p.analyticsErrors.Inc()
}

func (p *prometheusMetrics) RecordHealthCheckDuration(duration time.Duration) {
	p.healthCheckDur.Observe(duration.Seconds())
}

func (p *prometheusMetrics) RecordReadinessState(ready bool) {
	if ready {
		p.readinessState.Set(1)
	} else {
		p.readinessState.Set(0)
	}
}

func (p *prometheusMetrics) RecordStartupDuration(duration time.Duration) {
	p.startupDur.Set(duration.Seconds())
}

func (p *prometheusMetrics) HTTPHandler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}
