// Package constants defines application-wide immutable values.
package constants

// ContextKey defines custom type for context key safety.
type ContextKey string

const (
	// RequestIDKey is the context and header key used to track API requests.
	RequestIDKey ContextKey = "RequestID"

	// UserIDKey is the context key where authenticated user IDs are stored.
	UserIDKey ContextKey = "UserID"

	// AuthContextKey is the context key where the AuthContext struct is stored.
	AuthContextKey ContextKey = "AuthContext"

	// DefaultShortCodeLength is the fallback length for slug generation.
	DefaultShortCodeLength = 7
)

// ReservedAliases contains list of slug keywords that cannot be registered by users.
var ReservedAliases = []string{
	"api", "auth", "login", "logout", "register",
	"swagger", "health", "metrics", "admin", "users", "links",
}
