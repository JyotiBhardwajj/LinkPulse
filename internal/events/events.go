// Package events sets clean boundaries for publishing domain events asynchronously.
package events

import (
	"context"
	"linkpulse/internal/models"
)

// EventType represents the type of domain event.
type EventType string

const (
	// EventLinkClick is fired when a shortened URL is resolved and redirected.
	EventLinkClick EventType = "link.clicked"
)

// ClickEventPayload wraps the information sent to event consumers.
type ClickEventPayload struct {
	ShortCode string              `json:"short_code"`
	Details   models.ClickDetails `json:"details"`
}

// EventDispatcher defines the contract to dispatch system events.
type EventDispatcher interface {
	// Dispatch publishes an event payload to a queue or stream broker.
	Dispatch(ctx context.Context, eventType EventType, payload interface{}) error
}

type syncEventDispatcher struct{}

// NewSyncEventDispatcher creates a synchronous dispatcher placeholder.
func NewSyncEventDispatcher() EventDispatcher {
	return &syncEventDispatcher{}
}

// Dispatch executes a no-op dispatch (for Day 1 integration bounds).
func (d *syncEventDispatcher) Dispatch(ctx context.Context, eventType EventType, payload interface{}) error {
	// Real implementation would serialize and publish to Redis Streams / RabbitMQ / Kafka.
	return nil
}
