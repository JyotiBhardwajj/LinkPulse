// Package middleware defines Gin HTTP middlewares.
package middleware

import (
	"net/http"

	"linkpulse/internal/models"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// RequireRole checks if the authenticated user has one of the allowed roles.
func RequireRole(allowedRoles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := GetAuthContext(c)
		if !exists {
			utils.SendError(c, http.StatusUnauthorized, "Authentication context missing", "UNAUTHORIZED")
			c.Abort()
			return
		}

		roleAllowed := false
		for _, r := range allowedRoles {
			if authCtx.Role == r {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			utils.SendError(c, http.StatusForbidden, "Forbidden: insufficient permissions", "FORBIDDEN")
			c.Abort()
			return
		}

		c.Next()
	}
}
