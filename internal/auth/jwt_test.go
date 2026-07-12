package auth

import (
	"testing"
	"time"

	"linkpulse/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_AccessTokens(t *testing.T) {
	secret := "supersecretkeythatisreallylongandsecure"
	issuer := "test-issuer"
	userID := uuid.New()
	email := "test@example.com"
	ttl := 5 * time.Minute

	t.Run("Generate and validate valid token", func(t *testing.T) {
		token, err := GenerateAccessToken(userID, email, models.RoleUser, secret, ttl, issuer)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		claims, err := ValidateAccessToken(token, secret, issuer)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.Subject)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, models.RoleUser, claims.Role)
		assert.Equal(t, issuer, claims.Issuer)
		assert.NotEmpty(t, claims.ID)
	})

	t.Run("Expired token fails validation", func(t *testing.T) {
		// Generate with negative TTL
		token, err := GenerateAccessToken(userID, email, models.RoleUser, secret, -1*time.Minute, issuer)
		require.NoError(t, err)

		claims, err := ValidateAccessToken(token, secret, issuer)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "token validation failed")
	})

	t.Run("Wrong signing secret fails validation", func(t *testing.T) {
		token, err := GenerateAccessToken(userID, email, models.RoleUser, secret, ttl, issuer)
		require.NoError(t, err)

		claims, err := ValidateAccessToken(token, "wrongsecretkeythatisalsolongandsecure", issuer)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Wrong issuer fails validation", func(t *testing.T) {
		token, err := GenerateAccessToken(userID, email, models.RoleUser, secret, ttl, "some-other-issuer")
		require.NoError(t, err)

		claims, err := ValidateAccessToken(token, secret, issuer)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Invalid parsing format fails validation", func(t *testing.T) {
		claims, err := ValidateAccessToken("not.a.valid.token", secret, issuer)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}
