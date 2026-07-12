package models

import (
	"time"

	"github.com/google/uuid"
)

// CachedLink defines the cache storage format for a shortened link,
// keeping API DTOs and GORM model schemas isolated from infrastructure persistence layers.
type CachedLink struct {
	ID          uuid.UUID  `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}
