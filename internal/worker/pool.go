package worker

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"linkpulse/internal/repository"
)

// WorkerPool manages async processing of click analytics events.
type WorkerPool interface {
	// Start launches background worker goroutines.
	Start(ctx context.Context)

	// Submit attempts to enqueue a ClickEvent. Returns immediately.
	Submit(ctx context.Context, event ClickEvent) error

	// Shutdown stops all worker routines, waiting for queued tasks to finish up to context deadline.
	Shutdown(ctx context.Context) error
}

type workerPool struct {
	analyticsRepo repository.AnalyticsRepository
	workerCount   int
	queueSize     int
	queue         chan ClickEvent
	wg            sync.WaitGroup
	onceStart     sync.Once
	onceStop      sync.Once
	stopChan      chan struct{}
}

// NewWorkerPool instantiates a WorkerPool implementation.
func NewWorkerPool(analyticsRepo repository.AnalyticsRepository, workerCount, queueSize int) WorkerPool {
	if workerCount <= 0 {
		workerCount = 5
	}
	if queueSize <= 0 {
		queueSize = 1000
	}
	return &workerPool{
		analyticsRepo: analyticsRepo,
		workerCount:   workerCount,
		queueSize:     queueSize,
		queue:         make(chan ClickEvent, queueSize),
		stopChan:      make(chan struct{}),
	}
}

// Start launches concurrency worker loops.
func (p *workerPool) Start(ctx context.Context) {
	p.onceStart.Do(func() {
		slog.Info("Starting worker pool", "workers", p.workerCount, "queue_size", p.queueSize)
		for i := 0; i < p.workerCount; i++ {
			p.wg.Add(1)
			go p.workerLoop(ctx, i)
		}
	})
}

// Submit inserts event into buffered channel. Drops if queue is full.
func (p *workerPool) Submit(ctx context.Context, event ClickEvent) error {
	select {
	case p.queue <- event:
		return nil
	default:
		// Bounded channel overflow non-blocking policy
		slog.Warn("Worker queue is full, dropping click analytics event to preserve redirect latency",
			"link_id", event.LinkID,
			"timestamp", event.Timestamp,
		)
		return nil
	}
}

// Shutdown closes the queue channel and waits for workers to finish.
func (p *workerPool) Shutdown(ctx context.Context) error {
	var err error
	p.onceStop.Do(func() {
		slog.Info("Shutting down worker pool gracefully")
		close(p.stopChan)
		close(p.queue)

		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("Worker pool shut down successfully")
		case <-ctx.Done():
			err = fmt.Errorf("worker pool shutdown timed out: %w", ctx.Err())
			slog.Warn("Worker pool shutdown timed out before all events were processed")
		}
	})
	return err
}

func (p *workerPool) workerLoop(ctx context.Context, workerID int) {
	defer p.wg.Done()
	slog.Debug("Worker started", "worker_id", workerID)

	for {
		select {
		case <-p.stopChan:
			// Draining remaining items in channel after close
			for event := range p.queue {
				p.safeProcess(ctx, event)
			}
			return
		case event, ok := <-p.queue:
			if !ok {
				return
			}
			p.safeProcess(ctx, event)
		}
	}
}

func (p *workerPool) safeProcess(ctx context.Context, event ClickEvent) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Worker panic recovered",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	_ = processEvent(ctx, p.analyticsRepo, event)
}
