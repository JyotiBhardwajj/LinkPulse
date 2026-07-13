package worker

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"linkpulse/internal/metrics"
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

	// Ready checks if the worker pool is active and running.
	Ready(ctx context.Context) error
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
	started       bool
	stopped       bool
	mu            sync.Mutex
	metrics       metrics.Metrics
}

// NewWorkerPool instantiates a WorkerPool implementation.
func NewWorkerPool(analyticsRepo repository.AnalyticsRepository, workerCount, queueSize int, metricsTracker metrics.Metrics) WorkerPool {
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
		metrics:       metricsTracker,
	}
}

// Start launches concurrency worker loops.
func (p *workerPool) Start(ctx context.Context) {
	p.onceStart.Do(func() {
		p.mu.Lock()
		p.started = true
		p.mu.Unlock()
		slog.Info("Starting worker pool", "workers", p.workerCount, "queue_size", p.queueSize)

		p.metrics.RecordWorkerActive(p.workerCount)
		p.metrics.RecordWorkerQueueSize(0)

		for i := 0; i < p.workerCount; i++ {
			p.wg.Add(1)
			go p.workerLoop(ctx, i)
		}
	})
}

// Submit inserts event into buffered channel. Drops if queue is full.
func (p *workerPool) Submit(ctx context.Context, event ClickEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return fmt.Errorf("worker pool is stopped")
	}

	select {
	case p.queue <- event:
		p.metrics.RecordWorkerQueueSize(len(p.queue))
		return nil
	default:
		// Bounded channel overflow non-blocking policy
		slog.Warn("Worker queue is full, dropping click analytics event to preserve redirect latency",
			"link_id", event.LinkID,
			"timestamp", event.Timestamp,
		)
		p.metrics.RecordWorkerDropped()
		return nil
	}
}

// Shutdown closes the queue channel and waits for workers to finish.
func (p *workerPool) Shutdown(ctx context.Context) error {
	var err error
	p.onceStop.Do(func() {
		p.mu.Lock()
		p.stopped = true
		close(p.stopChan)
		close(p.queue)
		p.mu.Unlock()

		p.metrics.RecordWorkerActive(0)
		p.metrics.RecordWorkerQueueSize(0)

		slog.Info("Shutting down worker pool gracefully")

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

// Ready returns whether the worker pool is running and healthy.
func (p *workerPool) Ready(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		return fmt.Errorf("worker pool is not started")
	}
	if p.stopped {
		return fmt.Errorf("worker pool is stopped")
	}
	return nil
}

func (p *workerPool) workerLoop(ctx context.Context, workerID int) {
	defer p.wg.Done()
	slog.Debug("Worker started", "worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			// Draining remaining items in channel after close
			for event := range p.queue {
				p.metrics.RecordWorkerQueueSize(len(p.queue))
				p.safeProcess(ctx, event)
				p.metrics.RecordWorkerProcessed()
			}
			return
		case event, ok := <-p.queue:
			if !ok {
				return
			}
			p.metrics.RecordWorkerQueueSize(len(p.queue))
			p.safeProcess(ctx, event)
			p.metrics.RecordWorkerProcessed()
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

	_ = processEvent(ctx, p.analyticsRepo, event, p.metrics)
}
