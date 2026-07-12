package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkpulse/internal/auth"
	"linkpulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthAndRequireRoleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"

	r := gin.New()
	authMiddleware := Auth(secret, issuer)

	r.GET("/protected-user", authMiddleware, RequireRole(models.RoleUser), func(c *gin.Context) {
		authCtx, _ := GetAuthContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": authCtx.UserID.String(), "role": authCtx.Role})
	})

	r.GET("/protected-admin", authMiddleware, RequireRole(models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	userID := uuid.New()

	t.Run("Unauthorized if no token header provided", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected-user", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Succeeds for allowed user role", func(t *testing.T) {
		token, err := auth.GenerateAccessToken(userID, "user@example.com", models.RoleUser, secret, 5*time.Minute, issuer)
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected-user", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Forbidden (403) for user role accessing admin role", func(t *testing.T) {
		token, err := auth.GenerateAccessToken(userID, "user@example.com", models.RoleUser, secret, 5*time.Minute, issuer)
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected-admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Succeeds for admin role accessing admin endpoint", func(t *testing.T) {
		token, err := auth.GenerateAccessToken(userID, "admin@example.com", models.RoleAdmin, secret, 5*time.Minute, issuer)
		assert.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected-admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
