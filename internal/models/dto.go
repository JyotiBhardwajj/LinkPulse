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

