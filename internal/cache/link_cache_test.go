package cache

import (
	"context"
	"testing"
	"time"

	"linkpulse/internal/config"
	"linkpulse/internal/metrics"
	"linkpulse/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLinkCache_Integration(t *testing.T) {
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: "6379",
	}

	redisClient, err := NewRedisClient(cfg)
	if err != nil {
		t.Skip("Skipping Redis integration tests: localhost:6379 unreachable")
		return
	}
	defer func() { _ = redisClient.Close() }()

	ctx := context.Background()
	cache := NewLinkCache(redisClient, "test-link:", metrics.NewNoOpMetrics())

	shortCode := "test-code-" + uuid.New().String()[:8]
	cachedLink := &models.CachedLink{
		ID:          uuid.New(),
		OriginalURL: "https://stripe.com/docs",
		ShortCode:   shortCode,
		IsActive:    true,
	}

	t.Run("SetLink saves item and GetLink retrieves it", func(t *testing.T) {
		err := cache.SetLink(ctx, shortCode, cachedLink, 5*time.Second)
		assert.NoError(t, err)

		retrieved, err := cache.GetLink(ctx, shortCode)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, cachedLink.ID, retrieved.ID)
		assert.Equal(t, cachedLink.OriginalURL, retrieved.OriginalURL)
		assert.True(t, retrieved.IsActive)
	})

	t.Run("Exists confirms key presence", func(t *testing.T) {
		exists, err := cache.Exists(ctx, shortCode)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("DeleteLink invalidates cache key", func(t *testing.T) {
		err := cache.DeleteLink(ctx, shortCode)
		assert.NoError(t, err)

		retrieved, err := cache.GetLink(ctx, shortCode)
		assert.NoError(t, err)
		assert.Nil(t, retrieved)

		exists, err := cache.Exists(ctx, shortCode)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}
