package worker

import (
	"context"
	"log/slog"
	"time"

	"linkpulse/internal/metrics"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"

	"github.com/google/uuid"
)

// ClickEvent carries the click analytics event payload.
type ClickEvent struct {
	LinkID        uuid.UUID `json:"link_id"`
	Timestamp     time.Time `json:"timestamp"`
	UserAgent     string    `json:"user_agent"`
	Referrer      string    `json:"referrer"`
	IPAddressHash string    `json:"ip_address_hash"`
}

// processEvent maps ClickEvent to GORM Analytics model, validates properties, and persists to Postgres database.
func processEvent(ctx context.Context, repo repository.AnalyticsRepository, event ClickEvent, tracker metrics.Metrics) error {
	if event.LinkID == uuid.Nil {
		slog.Warn("Skipping event processing: LinkID is nil")
		return nil
	}

	analytics := &models.Analytics{
		ID:        uuid.New(),
		LinkID:    event.LinkID,
		ClickedAt: event.Timestamp,
		IPHash:    event.IPAddressHash,
		Country:   "Unknown",
		City:      "Unknown",
		Browser:   "Unknown",
		OS:        "Unknown",
		Device:    "Unknown",
		Referrer:  event.Referrer,
		UserAgent: event.UserAgent,
	}

	// 5-second timeout for database operation safety
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := repo.Create(dbCtx, analytics); err != nil {
		slog.Error("Failed to persist analytics record in background worker",
			"link_id", event.LinkID.String(),
			"error", err.Error(),
		)
		tracker.RecordAnalyticsError()
		return err
	}

	tracker.RecordAnalyticsWrite()
	return nil
}
