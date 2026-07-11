// Package constants defines application-wide immutable values.
package constants

// ContextKey defines custom type for context key safety.
type ContextKey string

const (
	// RequestIDKey is the context and header key used to track API requests.
	RequestIDKey ContextKey = "RequestID"

	// UserIDKey is the context key where authenticated user IDs are stored.
	UserIDKey ContextKey = "UserID"

	// DefaultShortCodeLength is the fallback length for slug generation.
	DefaultShortCodeLength = 7
)
