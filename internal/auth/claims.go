// Package auth handles authentication operations, JWT generation, and RTR.
package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserClaims defines custom claims payload signed within the JWT access token.
type UserClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// AuthContext represents the authenticated user session context injected into the request.
type AuthContext struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}
