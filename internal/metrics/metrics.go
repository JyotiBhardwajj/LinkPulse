// Package metrics provides observability and monitoring stubs.
package metrics

import "time"

// Metrics defines observability hooks for key business events.
type Metrics interface {
	// RecordLinkCreated tracks the total count of shortened links.
	RecordLinkCreated()

	// RecordLinkResolved tracks link redirection occurrences.
	RecordLinkResolved()

	// RecordRequestDuration logs the latency of HTTP operations.
	RecordRequestDuration(method, path string, duration time.Duration)
}

type noOpMetrics struct{}

// NewNoOpMetrics creates a no-operation metrics tracker to satisfy boundaries without overhead.
func NewNoOpMetrics() Metrics {
	return &noOpMetrics{}
}

func (m *noOpMetrics) RecordLinkCreated()                           {}
func (m *noOpMetrics) RecordLinkResolved()                          {}
func (m *noOpMetrics) RecordRequestDuration(string, string, time.Duration) {}
