package logger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogger_Lifecycle(t *testing.T) {
	// Initialize audit logger with a small queue size for testing
	loggerInstance := InitAuditLogger(10)
	ctx := context.Background()

	loggerInstance.Start(ctx)

	// Submit some records
	rec1 := AuditRecord{
		RequestID:  "req-1",
		UserID:     uuid.New(),
		Event:      EventLogin,
		Resource:   "users",
		ResourceID: "user-1",
	}

	loggerInstance.Submit(rec1)

	// Close logger with timeout context
	closeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := loggerInstance.Close(closeCtx)
	assert.NoError(t, err)
}

func TestAuditLogger_AutofillsAuditIDAndTimestamp(t *testing.T) {
	loggerInstance := &asyncAuditLogger{
		queue:    make(chan AuditRecord, 5),
		stopChan: make(chan struct{}),
	}

	rec := AuditRecord{
		RequestID: "req-2",
		Event:     EventLinkCreate,
	}

	loggerInstance.Submit(rec)

	select {
	case submitted := <-loggerInstance.queue:
		assert.NotEqual(t, uuid.Nil, submitted.AuditID)
		assert.False(t, submitted.Timestamp.IsZero())
	default:
		t.Fatal("Expected record in queue")
	}
}

func TestAuditLogger_DropsIfFull(t *testing.T) {
	loggerInstance := &asyncAuditLogger{
		queue:    make(chan AuditRecord, 1),
		stopChan: make(chan struct{}),
	}

	rec := AuditRecord{Event: EventLogin}

	// First should succeed
	loggerInstance.Submit(rec)

	// Second should be dropped immediately without blocking
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		loggerInstance.Submit(rec)
	}()

	// If it blocked, it would hang. Wait with timeout to ensure it doesn't block.
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("Submit blocked when queue was full")
	}
}
