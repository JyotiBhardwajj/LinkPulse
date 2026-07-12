package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMetrics_Isolation(t *testing.T) {
	// Create multiple instances of metrics tracker; should not cause registration panics
	m1, reg1 := NewPrometheusMetrics("linkpulse_test", "api_test")
	assert.NotNil(t, reg1)
	assert.NotNil(t, m1)

	m2, reg2 := NewPrometheusMetrics("linkpulse_test", "api_test")
	assert.NotNil(t, reg2)
	assert.NotNil(t, m2)

	// Verify they are completely separate instances
	assert.NotEqual(t, m1, m2)
}

func TestGetProductionMetrics_Singleton(t *testing.T) {
	// Calling GetProductionMetrics repeatedly should return the exact same instance
	m1, reg1 := GetProductionMetrics("linkpulse_prod_test", "api_prod_test")
	assert.NotNil(t, reg1)
	assert.NotNil(t, m1)

	m2, reg2 := GetProductionMetrics("linkpulse_prod_test", "api_prod_test")
	assert.NotNil(t, reg2)
	assert.NotNil(t, m2)

	assert.Same(t, m1, m2)
}

func TestPrometheusMetrics_RecordOperations(t *testing.T) {
	m, reg := NewPrometheusMetrics("linkpulse_op_test", "api_op_test")
	assert.NotNil(t, reg)

	// Verify recording does not panic on any metrics
	assert.NotPanics(t, func() {
		m.RecordHTTPRequest("GET", "/api/v1/links/:id", "200")
		m.RecordRequestDuration("GET", "/api/v1/links/:id", 123*time.Millisecond)
		m.RecordCacheHit()
		m.RecordCacheMiss()
		m.RecordCacheError()
		m.RecordWorkerProcessed()
		m.RecordWorkerDropped()
		m.RecordWorkerQueueSize(5)
		m.RecordWorkerActive(2)
		m.RecordLoginSuccess()
		m.RecordLoginFailure()
		m.RecordRefreshSuccess()
		m.RecordRefreshFailure()
		m.RecordLogout()
		m.RecordDBQuery("links", "query", 45*time.Millisecond)
		m.RecordLinkCreated()
		m.RecordLinkUpdated()
		m.RecordLinkDeleted()
		m.RecordLinkResolved()
		m.RecordAnalyticsWrite()
		m.RecordAnalyticsError()
	})
}
