package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"linkpulse/internal/metrics"

	"github.com/stretchr/testify/assert"
)

type mockChecker struct {
	name     string
	critical bool
	err      error
	delay    time.Duration
}

func (m *mockChecker) Name() string     { return m.name }
func (m *mockChecker) IsCritical() bool { return m.critical }
func (m *mockChecker) Check(ctx context.Context) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.err
}

func TestReadinessStateRace(t *testing.T) {
	m := metrics.NewNoOpMetrics()
	rs := NewReadinessState(m)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rs.SetReady()
			_ = rs.IsReady()
			rs.SetNotReady()
		}()
	}
	wg.Wait()
}

func TestHealthService_Checks(t *testing.T) {
	m := metrics.NewNoOpMetrics()
	rs := NewReadinessState(m)
	rs.SetReady()

	t.Run("Fully Healthy", func(t *testing.T) {
		hs := NewHealthService(rs, "1.0.0", 50*time.Millisecond, m)
		hs.Register(&mockChecker{name: "db", critical: true})
		hs.Register(&mockChecker{name: "redis", critical: false})

		resp, ready := hs.Ready(context.Background())
		assert.True(t, ready)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, 2, len(resp.Checks))
	})

	t.Run("Degraded (Optional Fails)", func(t *testing.T) {
		hs := NewHealthService(rs, "1.0.0", 50*time.Millisecond, m)
		hs.Register(&mockChecker{name: "db", critical: true})
		hs.Register(&mockChecker{name: "redis", critical: false, err: errors.New("redis down")})

		resp, ready := hs.Ready(context.Background())
		assert.True(t, ready) // Ready continues returning HTTP 200 (true) since it's optional
		assert.Equal(t, "degraded", resp.Status)
	})

	t.Run("Unhealthy (Critical Fails)", func(t *testing.T) {
		hs := NewHealthService(rs, "1.0.0", 50*time.Millisecond, m)
		hs.Register(&mockChecker{name: "db", critical: true, err: errors.New("db down")})
		hs.Register(&mockChecker{name: "redis", critical: false})

		resp, ready := hs.Ready(context.Background())
		assert.False(t, ready) // Fails overall readiness
		assert.Equal(t, "unhealthy", resp.Status)
	})

	t.Run("Checker Timeout Isolation", func(t *testing.T) {
		hs := NewHealthService(rs, "1.0.0", 20*time.Millisecond, m)
		hs.Register(&mockChecker{name: "db", critical: true, delay: 100 * time.Millisecond}) // exceeds timeout
		hs.Register(&mockChecker{name: "redis", critical: false})

		resp, ready := hs.Ready(context.Background())
		assert.False(t, ready)
		assert.Equal(t, "unhealthy", resp.Status)

		for _, check := range resp.Checks {
			if check.Name == "db" {
				assert.Equal(t, "unhealthy", check.Status)
				assert.Contains(t, check.Error, "context deadline exceeded")
			}
		}
	})

	t.Run("Startup Indicator Transitions", func(t *testing.T) {
		hs := NewHealthService(rs, "1.0.0", 50*time.Millisecond, m)
		assert.False(t, hs.Startup())
		hs.SetStartupComplete()
		assert.True(t, hs.Startup())
	})
}
