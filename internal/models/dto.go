// Package models defines the database schemas and DTO structs.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRegisterRequest represents the request body for user registration.
type UserRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// UserResponse represents the public user response.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ShortenLinkRequest represents the payload required to shorten a link.
type ShortenLinkRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	Title       string     `json:"title,omitempty" binding:"omitempty,max=255"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CustomSlug  string     `json:"custom_slug,omitempty" binding:"omitempty,alphanum,min=3,max=50"`
}

// LinkResponse represents the response containing shortened link details.
type LinkResponse struct {
	ID          uuid.UUID  `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	Title       string     `json:"title,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ClickDetails captures user-agent and geo-IP information for analytics extraction.
type ClickDetails struct {
	IPHash    string
	Country   string
	City      string
	Browser   string
	OS        string
	Device    string
	Referrer  string
	UserAgent string
}

// LinkStatsResponse contains click metrics grouped by dimensions.
type LinkStatsResponse struct {
	ID          uuid.UUID `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	TotalClicks int64     `json:"total_clicks"`
}

// ClickTimeMetric represents click aggregates collected in time buckets.
type ClickTimeMetric struct {
	TimeBucket time.Time `json:"time_bucket"`
	ClickCount int64     `json:"click_count"`
}

