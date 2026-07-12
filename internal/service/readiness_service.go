package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"linkpulse/internal/models"

	"golang.org/x/sync/errgroup"
)

// ReadinessChecker defines a contract for validating component health status.
type ReadinessChecker interface {
	Ready(ctx context.Context) error
}

// ReadinessService aggregates checks for all backend dependencies.
type ReadinessService interface {
	Check(ctx context.Context) (*models.ReadyResponse, error)
}

type readinessService struct {
	checkers map[string]ReadinessChecker
}

// NewReadinessService instantiates a ReadinessService with checkers.
func NewReadinessService(checkers map[string]ReadinessChecker) ReadinessService {
	return &readinessService{
		checkers: checkers,
	}
}

// Check queries all registered checkers in parallel using golang.org/x/sync/errgroup.
func (s *readinessService) Check(ctx context.Context) (*models.ReadyResponse, error) {
	// Enforce 2-second readiness timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	g, _ := errgroup.WithContext(timeoutCtx)

	var mu sync.Mutex
	states := map[string]string{
		"database":    "down",
		"redis":       "down",
		"worker_pool": "down",
	}

	for name, checker := range s.checkers {
		n := name
		ch := checker
		g.Go(func() error {
			// Pass timeoutCtx directly to avoid premature context cancellation across other checks
			err := ch.Ready(timeoutCtx)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				// Detail messages are logged to slog only (Sanitized HTTP responses)
				slog.Error("Readiness check failure",
					slog.String("dependency", n),
					slog.String("error", err.Error()),
				)
				states[n] = "down"
				return fmt.Errorf("dependency %s is not ready: %w", n, err)
			}

			states[n] = "up"
			return nil
		})
	}

	// Wait collects all parallel error checks
	err := g.Wait()

	status := "ready"
	if err != nil {
		status = "not_ready"
	}

	resp := &models.ReadyResponse{
		Status:     status,
		Database:   states["database"],
		Redis:      states["redis"],
		WorkerPool: states["worker_pool"],
		Timestamp:  time.Now().UTC(),
	}

	return resp, err
}
