// Package auth handles authentication operations, JWT generation, and RTR.
package auth

import (
	"errors"
	"fmt"
	"time"

	"linkpulse/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateAccessToken signs a new JWT access token containing standard claims, role, and a unique jti.
func GenerateAccessToken(userID uuid.UUID, email string, role models.Role, secret string, ttl time.Duration, issuer string) (string, error) {
	jti := uuid.New().String()
	now := time.Now()

	claims := UserClaims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenStr, nil
}

// ValidateAccessToken parses and validates a signed JWT access token.
func ValidateAccessToken(tokenStr string, secret string, issuer string) (*UserClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
	)

	var claims UserClaims
	token, err := parser.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		// Verify signature algorithm is exactly HS256
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return &claims, nil
}
