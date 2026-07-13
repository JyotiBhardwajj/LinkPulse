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

// AuthStatusResponse represents checks returned to determine authentication states.
type AuthStatusResponse struct {
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
	Page   int    `form:"page,default=1" binding:"omitempty,min=1"`
	Limit  int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Search string `form:"search" binding:"omitempty,max=255"`
	Sort   string `form:"sort,default=created_at" binding:"omitempty,oneof=created_at updated_at expires_at"`
	Order  string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
	Status string `form:"status" binding:"omitempty,oneof=active expired inactive deleted"`
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

// UserProfileResponse defines the payload returned for user details.
type UserProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionResponse defines the structure of an active login session.
type SessionResponse struct {
	SessionID      uuid.UUID `json:"session_id"`
	Device         string    `json:"device"`
	Browser        string    `json:"browser"`
	OS             string    `json:"os"`
	IPHash         string    `json:"ip_hash"`
	LastUsed       time.Time `json:"last_used"`
	CreatedAt      time.Time `json:"created_at"`
	CurrentSession bool      `json:"current_session"`
}

// ValidationError represents validation error details per field.
type ValidationError struct {
	Field   string `json:"field" example:"email"`
	Rule    string `json:"rule" example:"required"`
	Message string `json:"message" example:"The email field is required"`
}

// ErrorDetails defines backward compatible error properties.
type ErrorDetails struct {
	Code    string `json:"code" example:"INVALID_CREDENTIALS"`
	Message string `json:"message" example:"Invalid email or password"`
}

// SuccessResponse defines standard JSON wrapper.
type SuccessResponse struct {
	Success   bool        `json:"success" example:"true"`
	Message   string      `json:"message,omitempty" example:"Operation completed successfully"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty" example:"uuid-request-id"`
}

// ErrorResponse represents standardized RFC7807 problem details error payload.
type ErrorResponse struct {
	Success   bool          `json:"success" example:"false"`
	Error     *ErrorDetails `json:"error"`
	Type      string        `json:"type" example:"https://linkpulse.com/errors/invalid-credentials"`
	Title     string        `json:"title" example:"Invalid Credentials"`
	Status    int           `json:"status" example:"401"`
	Detail    string        `json:"detail" example:"The email or password provided is incorrect."`
	Instance  string        `json:"instance" example:"/api/v1/auth/login"`
	RequestID string        `json:"request_id" example:"uuid-request-id"`
}

// ValidationErrorResponse represents validation failure payload.
type ValidationErrorResponse struct {
	Success   bool              `json:"success" example:"false"`
	Error     *ErrorDetails     `json:"error"`
	Type      string            `json:"type" example:"https://linkpulse.com/errors/validation-error"`
	Title     string            `json:"title" example:"Validation Error"`
	Status    int               `json:"status" example:"422"`
	Detail    string            `json:"detail" example:"Validation failed"`
	Instance  string            `json:"instance" example:"/api/v1/links"`
	Details   []ValidationError `json:"details"`
	RequestID string            `json:"request_id" example:"uuid-request-id"`
}

// PaginationMetadata represents pagination offsets and boundaries.
type PaginationMetadata struct {
	Page        int   `json:"page" example:"1"`
	PageSize    int   `json:"page_size" example:"20"`
	Total       int64 `json:"total" example:"100"`
	TotalPages  int   `json:"total_pages" example:"5"`
	HasNext     bool  `json:"has_next" example:"true"`
	HasPrevious bool  `json:"has_previous" example:"false"`
}

// PaginatedResponse defines list wrappers.
type PaginatedResponse struct {
	Success   bool               `json:"success" example:"true"`
	Message   string             `json:"message,omitempty" example:"Items retrieved successfully"`
	Data      interface{}        `json:"data"`
	Metadata  PaginationMetadata `json:"metadata"`
	RequestID string             `json:"request_id" example:"uuid-request-id"`
}

// AnalyticsQueryRequest maps and validates range analytics parameters.
type AnalyticsQueryRequest struct {
	StartDate string `form:"start_date" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	EndDate   string `form:"end_date" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Interval  string `form:"interval,default=day" binding:"omitempty,oneof=hour day week month"`
	Limit     int    `form:"limit,default=10" binding:"omitempty,min=1,max=100"`
}
