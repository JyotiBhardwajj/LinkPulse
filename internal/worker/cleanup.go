package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"linkpulse/internal/service"
)

// CleanupScheduler manages scheduled background deactivations and cache invalidations for expired links.
type CleanupScheduler struct {
	linkService service.LinkService
	interval    time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
	onceStart   sync.Once
	onceStop    sync.Once
}

// NewCleanupScheduler creates a new CleanupScheduler.
func NewCleanupScheduler(linkService service.LinkService, interval time.Duration) *CleanupScheduler {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	return &CleanupScheduler{
		linkService: linkService,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

// Start launches the cleanup scheduler loop in a background goroutine.
func (s *CleanupScheduler) Start(ctx context.Context) {
	s.onceStart.Do(func() {
		s.wg.Add(1)
		go s.loop(ctx)
		slog.Info("Background cleanup scheduler started", "interval", s.interval)
	})
}

// Stop halts the scheduler loop gracefully.
func (s *CleanupScheduler) Stop() {
	s.onceStop.Do(func() {
		slog.Info("Stopping background cleanup scheduler")
		close(s.stopChan)
		s.wg.Wait()
		slog.Info("Background cleanup scheduler stopped successfully")
	})
}

func (s *CleanupScheduler) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.runCleanup(ctx)
		}
	}
}

func (s *CleanupScheduler) runCleanup(ctx context.Context) {
	slog.Info("Running scheduled expired links cleanup job")
	// Set 2-minute safety limit for execution context
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	count, err := s.linkService.DeactivateExpiredLinks(runCtx)
	if err != nil {
		slog.Error("Scheduled expired links cleanup job failed", "error", err.Error())
		return
	}

	if count > 0 {
		slog.Info("Deactivated expired links successfully", "deactivated_count", count)
	} else {
		slog.Debug("Cleanup job finished: no expired links identified")
	}
}
