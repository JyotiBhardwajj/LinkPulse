package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type spyMetrics struct {
	lastMethod string
	lastRoute  string
	lastStatus string
	durations  int
}

func (s *spyMetrics) RecordHTTPRequest(method, route, status string) {
	s.lastMethod = method
	s.lastRoute = route
	s.lastStatus = status
}

func (s *spyMetrics) RecordRequestDuration(method, route string, duration time.Duration) {
	s.durations++
}

func (s *spyMetrics) RecordCacheHit()                                                    {}
func (s *spyMetrics) RecordCacheMiss()                                                   {}
func (s *spyMetrics) RecordCacheError()                                                  {}
func (s *spyMetrics) RecordWorkerProcessed()                                             {}
func (s *spyMetrics) RecordWorkerDropped()                                               {}
func (s *spyMetrics) RecordWorkerQueueSize(size int)                                     {}
func (s *spyMetrics) RecordWorkerActive(count int)                                       {}
func (s *spyMetrics) RecordLoginSuccess()                                                {}
func (s *spyMetrics) RecordLoginFailure()                                                {}
func (s *spyMetrics) RecordRefreshSuccess()                                              {}
func (s *spyMetrics) RecordRefreshFailure()                                              {}
func (s *spyMetrics) RecordLogout()                                                      {}
func (s *spyMetrics) RecordDBQuery(repository, operation string, duration time.Duration) {}
func (s *spyMetrics) RecordLinkCreated()                                                 {}
func (s *spyMetrics) RecordLinkUpdated()                                                 {}
func (s *spyMetrics) RecordLinkDeleted()                                                 {}
func (s *spyMetrics) RecordLinkResolved()                                                {}
func (s *spyMetrics) RecordAnalyticsWrite()                                              {}
func (s *spyMetrics) RecordAnalyticsError()                                              {}

func TestMetricsMiddleware_RouteMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyMetrics{}

	r := gin.New()
	r.Use(MetricsMiddleware(spy))

	r.GET("/links/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 1. Dynamic route /links/123
	req := httptest.NewRequest(http.MethodGet, "/links/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "GET", spy.lastMethod)
	assert.Equal(t, "/links/:id", spy.lastRoute)
	assert.Equal(t, "200", spy.lastStatus)

	// 2. Dynamic route /links/abc
	req2 := httptest.NewRequest(http.MethodGet, "/links/abc", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "/links/:id", spy.lastRoute) // Checks route template grouping is preserved
}

func TestMetricsMiddleware_UnknownRouteFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyMetrics{}

	r := gin.New()
	r.Use(MetricsMiddleware(spy))

	// No route is registered -> 404
	req := httptest.NewRequest(http.MethodGet, "/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "unknown", spy.lastRoute) // Empty FullPath() falls back to "unknown"
}

func TestMetricsMiddleware_MetricsBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyMetrics{}

	r := gin.New()
	r.Use(MetricsMiddleware(spy))

	r.GET("/metrics", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, spy.lastRoute) // Verify no metric was recorded for /metrics path
	assert.Equal(t, 0, spy.durations)
}
