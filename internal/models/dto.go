// Package models defines the database schemas and DTO structs.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRegisterRequest represents the request body for user registration.
// Note: password maximum length is set to 72 characters because bcrypt ignores any bytes past 72.
type UserRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest represents credentials supplied for authentication checks.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents standard OAuth2-like JWT output payload.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access token TTL in seconds
}

// RefreshRequest is the payload required to rotate user sessions.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// SessionResponse represents checks returned to determine authentication states.
type SessionResponse struct {
	Authenticated bool         `json:"authenticated"`
	User          UserResponse `json:"user"`
}

// UserResponse represents the public user response.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateLinkRequest represents the payload required to shorten a link.
type CreateLinkRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	Title       string     `json:"title,omitempty" binding:"omitempty,max=255"`
	CustomAlias string     `json:"custom_alias,omitempty" binding:"omitempty,min=3,max=50"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// UpdateLinkRequest represents the payload to modify an existing link's settings.
type UpdateLinkRequest struct {
	Title     *string    `json:"title,omitempty" binding:"omitempty,max=255"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
}

// LinkResponse represents the response containing shortened link details.
type LinkResponse struct {
	ID          uuid.UUID  `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	Title       string     `json:"title,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	ClickCount  int64      `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListLinksQuery defines filter search sorting and paging variables.
type ListLinksQuery struct {
	Page   int    `form:"page,default=1"`
	Limit  int    `form:"limit,default=20"`
	Search string `form:"search"`
	Sort   string `form:"sort,default=created_at"`
	Order  string `form:"order,default=desc"`
	Status string `form:"status"` // active, expired, inactive, deleted
}

// PaginationResponse represents a generic paginated output envelope.
type PaginationResponse[T any] struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	Items      []T   `json:"items"`
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

// AnalyticsOverview represents global statistics for a user's link portfolio.
type AnalyticsOverview struct {
	TotalLinks       int64 `json:"total_links"`
	ActiveLinks      int64 `json:"active_links"`
	InactiveLinks    int64 `json:"inactive_links"`
	TotalClicks      int64 `json:"total_clicks"`
	TodayClicks      int64 `json:"today_clicks"`
	Last7DaysClicks  int64 `json:"last_7_days_clicks"`
	Last30DaysClicks int64 `json:"last_30_days_clicks"`
}

// ClickTimeSeriesPoint represents a single data point in a click timeline.
type ClickTimeSeriesPoint struct {
	Timestamp string `json:"timestamp"`
	Clicks    int64  `json:"clicks"`
}

// TopLinkMetric represents a top performing shortened link.
type TopLinkMetric struct {
	ShortCode     string     `json:"short_code"`
	OriginalURL   string     `json:"original_url"`
	ClickCount    int64      `json:"click_count"`
	LastClickedAt *time.Time `json:"last_clicked_at,omitempty"`
}

// DistributionItem represents a percentage/count tuple.
type DistributionItem struct {
	Name       string  `json:"name"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// LinkAnalyticsResponse represents the full analytical report of a single link.
type LinkAnalyticsResponse struct {
	LinkID              uuid.UUID              `json:"link_id"`
	OriginalURL         string                 `json:"original_url"`
	ShortCode           string                 `json:"short_code"`
	TotalClicks         int64                  `json:"total_clicks"`
	ClicksOverTime      []ClickTimeSeriesPoint `json:"clicks_over_time"`
	BrowserDistribution []DistributionItem     `json:"browser_distribution"`
	DeviceDistribution  []DistributionItem     `json:"device_distribution"`
	TopReferrers        []DistributionItem     `json:"top_referrers"`
}

// HealthResponse represents liveness API structure.
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	GitCommit string    `json:"git_commit"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadyResponse represents readiness API structure.
type ReadyResponse struct {
	Status     string    `json:"status"`
	Database   string    `json:"database"`
	Redis      string    `json:"redis"`
	WorkerPool string    `json:"worker_pool"`
	Timestamp  time.Time `json:"timestamp"`
}

