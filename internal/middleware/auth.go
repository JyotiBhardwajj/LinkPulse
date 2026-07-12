// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"linkpulse/internal/auth"
	"linkpulse/internal/constants"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Auth parses and validates a Bearer JWT, injecting the AuthContext into the request.
func Auth(secret string, issuer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.SendError(c, http.StatusUnauthorized, "Authorization header is required", "UNAUTHORIZED")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.SendError(c, http.StatusUnauthorized, "Authorization header format must be Bearer <token>", "UNAUTHORIZED")
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := auth.ValidateAccessToken(tokenStr, secret, issuer)
		if err != nil {
			utils.SendError(c, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			utils.SendError(c, http.StatusUnauthorized, "invalid token subject payload", "UNAUTHORIZED")
			c.Abort()
			return
		}

		sessionID, _ := uuid.Parse(claims.ID)
		authCtx := auth.AuthContext{
			UserID:    userID,
			Email:     claims.Email,
			Role:      claims.Role,
			SessionID: sessionID,
		}

		// Inject in Gin context
		c.Set(string(constants.AuthContextKey), authCtx)

		// Inject in standard request context for down-stream service context propagation
		ctx := context.WithValue(c.Request.Context(), constants.AuthContextKey, authCtx)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetAuthContext extracts the AuthContext from the Gin context.
func GetAuthContext(c *gin.Context) (auth.AuthContext, bool) {
	if val, exists := c.Get(string(constants.AuthContextKey)); exists {
		if authCtx, ok := val.(auth.AuthContext); ok {
			return authCtx, true
		}
	}
	return auth.AuthContext{}, false
}
