// Package metrics provides observability and monitoring stubs.
package metrics

import "time"

// Metrics defines observability hooks for key business events.
type Metrics interface {
	RecordHTTPRequest(method, route, status string)
	RecordRequestDuration(method, route string, duration time.Duration)
	RecordCacheHit()
	RecordCacheMiss()
	RecordCacheError()
	RecordWorkerProcessed()
	RecordWorkerDropped()
	RecordWorkerQueueSize(size int)
	RecordWorkerActive(count int)
	RecordLoginSuccess()
	RecordLoginFailure()
	RecordRefreshSuccess()
	RecordRefreshFailure()
	RecordLogout()
	RecordDBQuery(repository, operation string, duration time.Duration)
	RecordLinkCreated()
	RecordLinkUpdated()
	RecordLinkDeleted()
	RecordLinkResolved()
	RecordAnalyticsWrite()
	RecordAnalyticsError()

	// Day 10 extensions
	RecordHealthCheckDuration(duration time.Duration)
	RecordReadinessState(ready bool)
	RecordStartupDuration(duration time.Duration)
}

type noOpMetrics struct{}

// NewNoOpMetrics creates a no-operation metrics tracker to satisfy boundaries without overhead.
func NewNoOpMetrics() Metrics {
	return &noOpMetrics{}
}

func (m *noOpMetrics) RecordHTTPRequest(method, route, status string)                     {}
func (m *noOpMetrics) RecordRequestDuration(method, route string, duration time.Duration) {}
func (m *noOpMetrics) RecordCacheHit()                                                    {}
func (m *noOpMetrics) RecordCacheMiss()                                                   {}
func (m *noOpMetrics) RecordCacheError()                                                  {}
func (m *noOpMetrics) RecordWorkerProcessed()                                             {}
func (m *noOpMetrics) RecordWorkerDropped()                                               {}
func (m *noOpMetrics) RecordWorkerQueueSize(size int)                                     {}
func (m *noOpMetrics) RecordWorkerActive(count int)                                       {}
func (m *noOpMetrics) RecordLoginSuccess()                                                {}
func (m *noOpMetrics) RecordLoginFailure()                                                {}
func (m *noOpMetrics) RecordRefreshSuccess()                                              {}
func (m *noOpMetrics) RecordRefreshFailure()                                              {}
func (m *noOpMetrics) RecordLogout()                                                      {}
func (m *noOpMetrics) RecordDBQuery(repository, operation string, duration time.Duration) {}
func (m *noOpMetrics) RecordLinkCreated()                                                 {}
func (m *noOpMetrics) RecordLinkUpdated()                                                 {}
func (m *noOpMetrics) RecordLinkDeleted()                                                 {}
func (m *noOpMetrics) RecordLinkResolved()                                                {}
func (m *noOpMetrics) RecordAnalyticsWrite()                                              {}
func (m *noOpMetrics) RecordAnalyticsError()                                              {}

func (m *noOpMetrics) RecordHealthCheckDuration(duration time.Duration) {}
func (m *noOpMetrics) RecordReadinessState(ready bool)                  {}
func (m *noOpMetrics) RecordStartupDuration(duration time.Duration)     {}
