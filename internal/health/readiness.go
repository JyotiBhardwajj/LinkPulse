package health

import (
	"sync/atomic"

	"linkpulse/internal/metrics"
)

// ReadinessState tracks the application's overall traffic readiness.
type ReadinessState struct {
	ready   atomic.Bool
	metrics metrics.Metrics
}

// NewReadinessState initializes a new ReadinessState.
func NewReadinessState(m metrics.Metrics) *ReadinessState {
	rs := &ReadinessState{
		metrics: m,
	}
	rs.SetNotReady()
	return rs
}

// SetReady sets the status to ready (true) and records the metric.
func (r *ReadinessState) SetReady() {
	r.ready.Store(true)
	r.metrics.RecordReadinessState(true)
}

// SetNotReady sets the status to not ready (false) and records the metric.
func (r *ReadinessState) SetNotReady() {
	r.ready.Store(false)
	r.metrics.RecordReadinessState(false)
}

// IsReady returns the current readiness status.
func (r *ReadinessState) IsReady() bool {
	return r.ready.Load()
}
