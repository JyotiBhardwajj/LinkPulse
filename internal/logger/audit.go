// Package logger manages structured standard logs and async audit events.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuditEvent defines a strongly-typed string alias for audit events.
type AuditEvent string

const (
	EventLogin      AuditEvent = "login"
	EventLogout     AuditEvent = "logout"
	EventLogoutAll  AuditEvent = "logout_all"
	EventRefresh    AuditEvent = "refresh"
	EventRoleChange AuditEvent = "role_change"
	EventLinkCreate AuditEvent = "link_create"
	EventLinkUpdate AuditEvent = "link_update"
	EventLinkDelete AuditEvent = "link_delete"
)

// AuditRecord represents the audit event logging payload.
type AuditRecord struct {
	AuditID    uuid.UUID  `json:"audit_id"`
	RequestID  string     `json:"request_id"`
	UserID     uuid.UUID  `json:"user_id"`
	Event      AuditEvent `json:"event"`
	Resource   string     `json:"resource"`
	ResourceID string     `json:"resource_id"`
	IPHash     string     `json:"ip_hash"`
	Timestamp  time.Time  `json:"timestamp"`
}

// AsyncAuditLogger defines the contract for async audit logging with lifecycle controls.
type AsyncAuditLogger interface {
	Start(ctx context.Context)
	Submit(rec AuditRecord)
	Close(ctx context.Context) error
}

type asyncAuditLogger struct {
	queue     chan AuditRecord
	stopChan  chan struct{}
	wg        sync.WaitGroup
	onceStart sync.Once
	onceStop  sync.Once
	started   bool
	stopped   bool
	mu        sync.Mutex
}

var (
	globalAuditLogger AsyncAuditLogger
	auditLoggerMu     sync.RWMutex
)

// InitAuditLogger instantiates the global audit logger worker.
func InitAuditLogger(queueSize int) AsyncAuditLogger {
	if queueSize <= 0 {
		queueSize = 1000
	}
	auditLoggerMu.Lock()
	globalAuditLogger = &asyncAuditLogger{
		queue:    make(chan AuditRecord, queueSize),
		stopChan: make(chan struct{}),
	}
	logger := globalAuditLogger
	auditLoggerMu.Unlock()
	return logger
}

// GetAuditLogger returns the global initialized audit logger instance, falling back to a no-op implementation if uninitialized.
func GetAuditLogger() AsyncAuditLogger {
	auditLoggerMu.RLock()
	logger := globalAuditLogger
	auditLoggerMu.RUnlock()
	if logger == nil {
		return &noopAuditLogger{}
	}
	return logger
}

type noopAuditLogger struct{}

func (n *noopAuditLogger) Start(ctx context.Context) {}
func (n *noopAuditLogger) Submit(rec AuditRecord)    {}
func (n *noopAuditLogger) Close(ctx context.Context) error {
	return nil
}

// Start launches the background processing worker loop.
func (l *asyncAuditLogger) Start(ctx context.Context) {
	l.onceStart.Do(func() {
		l.mu.Lock()
		l.started = true
		l.mu.Unlock()

		l.wg.Add(1)
		go l.workerLoop(ctx)
	})
}

// Submit enqueues an audit event to be logged asynchronously. Drops event if queue is full.
func (l *asyncAuditLogger) Submit(rec AuditRecord) {
	l.mu.Lock()
	isStopped := l.stopped
	l.mu.Unlock()

	if isStopped {
		slog.Warn("Audit logger is closed, ignoring audit submission", slog.String("event", string(rec.Event)))
		return
	}

	// Generate a unique AuditID for this event before enqueueing
	if rec.AuditID == uuid.Nil {
		rec.AuditID = uuid.New()
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().UTC()
	}

	select {
	case l.queue <- rec:
	default:
		slog.Warn("Audit log queue is full, dropping event to prevent blocking API request",
			slog.String("event", string(rec.Event)),
			slog.String("user_id", rec.UserID.String()),
		)
	}
}

// Close stops accepting new events, drains the queue, and shuts down the background worker.
func (l *asyncAuditLogger) Close(ctx context.Context) error {
	var err error
	l.onceStop.Do(func() {
		l.mu.Lock()
		l.stopped = true
		l.mu.Unlock()

		// Reset globalAuditLogger so subsequent tests fallback to no-op audit logger
		auditLoggerMu.Lock()
		if globalAuditLogger == l {
			globalAuditLogger = nil
		}
		auditLoggerMu.Unlock()

		slog.Info("Closing audit logger gracefully, draining remaining records")
		close(l.stopChan)

		done := make(chan struct{})
		go func() {
			l.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("Audit logger drained and closed successfully")
		case <-ctx.Done():
			err = fmt.Errorf("audit logger close timed out: %w", ctx.Err())
			slog.Warn("Audit logger close timed out before all events were processed")
		}
	})
	return err
}

func (l *asyncAuditLogger) workerLoop(ctx context.Context) {
	defer l.wg.Done()

	for {
		select {
		case <-l.stopChan:
			// Drain remaining events in channel non-blockingly
			for {
				select {
				case rec := <-l.queue:
					l.flushRecord(rec)
				default:
					return
				}
			}
		case rec := <-l.queue:
			l.flushRecord(rec)
		}
	}
}

// flushRecord writes the audit log to slog output.
func (l *asyncAuditLogger) flushRecord(rec AuditRecord) {
	slog.Info("Audit Log Event",
		slog.String("audit_id", rec.AuditID.String()),
		slog.String("request_id", rec.RequestID),
		slog.String("user_id", rec.UserID.String()),
		slog.String("event", string(rec.Event)),
		slog.String("resource", rec.Resource),
		slog.String("resource_id", rec.ResourceID),
		slog.String("timestamp", rec.Timestamp.Format(time.RFC3339)),
		slog.String("ip_hash", rec.IPHash),
	)
}
