package health

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"linkpulse/internal/metrics"
)

// DependencyResult represents the health check status of a single dependency.
type DependencyResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
	Critical   bool   `json:"critical"`
}

// HealthResponse represents the aggregated health check response.
type HealthResponse struct {
	Status     string             `json:"status"` // "healthy", "degraded", or "unhealthy"
	Timestamp  string             `json:"timestamp"`
	DurationMS int64              `json:"duration_ms"`
	Version    string             `json:"version"`
	Checks     []DependencyResult `json:"checks"`
}

// HealthService manages registration and execution of parallel health checks.
type HealthService interface {
	Register(checker Checker)
	CheckAll(ctx context.Context) HealthResponse
	Live() bool
	Ready(ctx context.Context) (HealthResponse, bool)
	Startup() bool
	SetStartupComplete()
}

type healthService struct {
	mu              sync.RWMutex
	checkers        []Checker
	readinessState  *ReadinessState
	startupComplete atomic.Bool
	version         string
	defaultTimeout  time.Duration
	metrics         metrics.Metrics
}

// NewHealthService initializes a new instance of HealthService.
func NewHealthService(rs *ReadinessState, version string, defaultTimeout time.Duration, m metrics.Metrics) HealthService {
	return &healthService{
		readinessState: rs,
		version:        version,
		defaultTimeout: defaultTimeout,
		metrics:        m,
	}
}

func (s *healthService) Register(checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers = append(s.checkers, checker)
}

func (s *healthService) CheckAll(ctx context.Context) HealthResponse {
	s.mu.RLock()
	checkers := make([]Checker, len(s.checkers))
	copy(checkers, s.checkers)
	s.mu.RUnlock()

	start := time.Now()
	var wg sync.WaitGroup
	resultsChan := make(chan DependencyResult, len(checkers))

	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			checkStart := time.Now()

			// isolated timeout per dependency check
			checkCtx, cancel := context.WithTimeout(ctx, s.defaultTimeout)
			defer cancel()

			errChan := make(chan error, 1)
			go func() {
				errChan <- c.Check(checkCtx)
			}()

			var err error
			select {
			case err = <-errChan:
			case <-checkCtx.Done():
				err = checkCtx.Err()
			}

			checkDuration := time.Since(checkStart)
			s.metrics.RecordHealthCheckDuration(checkDuration)

			status := "healthy"
			errStr := ""
			if err != nil {
				status = "unhealthy"
				errStr = err.Error()
			}

			resultsChan <- DependencyResult{
				Name:       c.Name(),
				Status:     status,
				DurationMS: checkDuration.Milliseconds(),
				Error:      errStr,
				Critical:   c.IsCritical(),
			}
		}(checker)
	}

	wg.Wait()
	close(resultsChan)

	hasCriticalFailure := false
	hasOptionalFailure := false
	checks := make([]DependencyResult, 0, len(checkers))

	for res := range resultsChan {
		if res.Status != "healthy" {
			if res.Critical {
				hasCriticalFailure = true
			} else {
				hasOptionalFailure = true
			}
		}
		checks = append(checks, res)
	}

	status := "healthy"
	if hasCriticalFailure {
		status = "unhealthy"
	} else if hasOptionalFailure {
		status = "degraded"
	}

	return HealthResponse{
		Status:     status,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		DurationMS: time.Since(start).Milliseconds(),
		Version:    s.version,
		Checks:     checks,
	}
}

func (s *healthService) Live() bool {
	// GET /health/live checks only whether the process is alive.
	return true
}

func (s *healthService) Ready(ctx context.Context) (HealthResponse, bool) {
	resp := s.CheckAll(ctx)

	// Returns HTTP 503 if global readinessState is false or any critical dependency fails
	if !s.readinessState.IsReady() || resp.Status == "unhealthy" {
		return resp, false
	}
	return resp, true
}

func (s *healthService) Startup() bool {
	return s.startupComplete.Load()
}

func (s *healthService) SetStartupComplete() {
	s.startupComplete.Store(true)
}
