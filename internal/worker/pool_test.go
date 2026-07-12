package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"linkpulse/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockAnalyticsRepo struct {
	mu      sync.Mutex
	clicks  []*models.Analytics
	panics  bool
	blockChan chan struct{}
}

func (m *mockAnalyticsRepo) Create(ctx context.Context, click *models.Analytics) error {
	if m.blockChan != nil {
		<-m.blockChan
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.panics {
		panic("mock repository panic")
	}
	m.clicks = append(m.clicks, click)
	return nil
}

func (m *mockAnalyticsRepo) GetClicksCount(ctx context.Context, linkID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockAnalyticsRepo) GetBrowserDistribution(ctx context.Context, linkID uuid.UUID) (map[string]int64, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetClicksOverTime(ctx context.Context, linkID uuid.UUID, interval string) ([]models.ClickTimeMetric, error) {
	return nil, nil
}

func TestWorkerPool_Processing(t *testing.T) {
	repo := &mockAnalyticsRepo{}
	pool := NewWorkerPool(repo, 3, 10)
	pool.Start(context.Background())
	defer func() { _ = pool.Shutdown(context.Background()) }()

	ctx := context.Background()
	linkID := uuid.New()

	t.Run("Submit successfully processes ClickEvent", func(t *testing.T) {
		event := ClickEvent{
			LinkID:        linkID,
			Timestamp:     time.Now(),
			UserAgent:     "test-agent",
			Referrer:      "test-ref",
			IPAddressHash: "hash-hash",
		}

		err := pool.Submit(ctx, event)
		assert.NoError(t, err)

		// Wait briefly for worker goroutine execution
		time.Sleep(100 * time.Millisecond)

		repo.mu.Lock()
		count := len(repo.clicks)
		repo.mu.Unlock()

		assert.Equal(t, 1, count)
	})
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	repo := &mockAnalyticsRepo{panics: true}
	pool := NewWorkerPool(repo, 1, 10)
	pool.Start(context.Background())
	defer func() { _ = pool.Shutdown(context.Background()) }()

	ctx := context.Background()
	event := ClickEvent{
		LinkID:        uuid.New(),
		Timestamp:     time.Now(),
		UserAgent:     "panic-agent",
		IPAddressHash: "panic-hash",
	}

	err := pool.Submit(ctx, event)
	assert.NoError(t, err)

	// Wait briefly for panic recovery to execute
	time.Sleep(100 * time.Millisecond)
	// If the worker pool did not crash, panic recovery succeeded.
}

func TestWorkerPool_QueueOverflow(t *testing.T) {
	// A blocked repository to fill up the queue
	block := make(chan struct{})
	repo := &mockAnalyticsRepo{blockChan: block}

	// 1 worker, queue capacity 2
	pool := NewWorkerPool(repo, 1, 2)
	pool.Start(context.Background())

	ctx := context.Background()

	// Fill the worker execution slot + the queue (capacity 2)
	_ = pool.Submit(ctx, ClickEvent{LinkID: uuid.New()}) // occupying worker
	_ = pool.Submit(ctx, ClickEvent{LinkID: uuid.New()}) // occupying queue slot 1
	_ = pool.Submit(ctx, ClickEvent{LinkID: uuid.New()}) // occupying queue slot 2

	// This event should overflow and be dropped immediately (returns nil, does not block)
	start := time.Now()
	err := pool.Submit(ctx, ClickEvent{LinkID: uuid.New()})
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, 50*time.Millisecond) // Verify it did not block

	// Cleanup
	close(block)
	_ = pool.Shutdown(context.Background())
}
