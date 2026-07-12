// Package cache manages Redis client setup and application-specific cache layers.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"linkpulse/internal/metrics"
	"linkpulse/internal/models"

	"github.com/redis/go-redis/v9"
)

// LinkCache defines caching operations for shortened URL responses.
type LinkCache interface {
	// GetLink retrieves the CachedLink representation from cache.
	GetLink(ctx context.Context, shortCode string) (*models.CachedLink, error)

	// SetLink persists the CachedLink mapping into cache using configured prefix and calculated TTL.
	SetLink(ctx context.Context, shortCode string, link *models.CachedLink, ttl time.Duration) error

	// DeleteLink removes a cached mapping immediately.
	DeleteLink(ctx context.Context, shortCode string) error

	// Exists checks if a cached mapping key exists.
	Exists(ctx context.Context, shortCode string) (bool, error)
}

type linkCache struct {
	redisClient *RedisClient
	prefix      string
	metrics     metrics.Metrics
}

// NewLinkCache instantiates a LinkCache implementation backed by Redis.
func NewLinkCache(redisClient *RedisClient, prefix string, metrics metrics.Metrics) LinkCache {
	return &linkCache{
		redisClient: redisClient,
		prefix:      prefix,
		metrics:     metrics,
	}
}

// GetLink checks the Redis cache for the given short code, deserializing JSON.
func (c *linkCache) GetLink(ctx context.Context, shortCode string) (*models.CachedLink, error) {
	start := time.Now()
	key := c.prefix + shortCode

	val, err := c.redisClient.Client.Get(ctx, key).Result()
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Info("Cache Miss",
				"operation", "GetLink",
				"short_code", shortCode,
				"duration_ms", float64(duration.Microseconds())/1000.0,
				"status", "miss",
			)
			c.metrics.RecordCacheMiss()
			return nil, nil // Return nil, nil on cache miss
		}

		slog.Error("Cache Error",
			"operation", "GetLink",
			"short_code", shortCode,
			"duration_ms", float64(duration.Microseconds())/1000.0,
			"status", "error",
			"error", err.Error(),
		)
		c.metrics.RecordCacheError()
		return nil, err
	}

	var cachedLink models.CachedLink
	if err := json.Unmarshal([]byte(val), &cachedLink); err != nil {
		slog.Error("Cache Deserialization Error",
			"operation", "GetLink",
			"short_code", shortCode,
			"duration_ms", float64(duration.Microseconds())/1000.0,
			"status", "error",
			"error", err.Error(),
		)
		c.metrics.RecordCacheError()
		return nil, err
	}

	slog.Info("Cache Hit",
		"operation", "GetLink",
		"short_code", shortCode,
		"duration_ms", float64(duration.Microseconds())/1000.0,
		"status", "hit",
	)
	c.metrics.RecordCacheHit()
	return &cachedLink, nil
}

// SetLink saves the short code to CachedLink mapping in Redis using calculated TTL.
func (c *linkCache) SetLink(ctx context.Context, shortCode string, link *models.CachedLink, ttl time.Duration) error {
	start := time.Now()
	key := c.prefix + shortCode

	bytes, err := json.Marshal(link)
	if err != nil {
		slog.Error("Cache Serialization Error",
			"operation", "SetLink",
			"short_code", shortCode,
			"duration_ms", 0.0,
			"status", "error",
			"error", err.Error(),
		)
		c.metrics.RecordCacheError()
		return err
	}

	err = c.redisClient.Client.Set(ctx, key, string(bytes), ttl).Err()
	duration := time.Since(start)

	if err != nil {
		slog.Error("Cache Error",
			"operation", "SetLink",
			"short_code", shortCode,
			"duration_ms", float64(duration.Microseconds())/1000.0,
			"status", "error",
			"error", err.Error(),
		)
		c.metrics.RecordCacheError()
		return err
	}

	slog.Info("Cache Write",
		"operation", "SetLink",
		"short_code", shortCode,
		"duration_ms", float64(duration.Microseconds())/1000.0,
		"status", "success",
	)
	return nil
}

// DeleteLink removes the cached mapping from Redis.
func (c *linkCache) DeleteLink(ctx context.Context, shortCode string) error {
	start := time.Now()
	key := c.prefix + shortCode

	err := c.redisClient.Client.Del(ctx, key).Err()
	duration := time.Since(start)

	if err != nil {
		slog.Error("Cache Error",
			"operation", "DeleteLink",
			"short_code", shortCode,
			"duration_ms", float64(duration.Microseconds())/1000.0,
			"status", "error",
			"error", err.Error(),
		)
		c.metrics.RecordCacheError()
		return err
	}

	slog.Info("Cache Delete",
		"operation", "DeleteLink",
		"short_code", shortCode,
		"duration_ms", float64(duration.Microseconds())/1000.0,
		"status", "success",
	)
	return nil
}

// Exists checks if the cached key exists in Redis.
func (c *linkCache) Exists(ctx context.Context, shortCode string) (bool, error) {
	key := c.prefix + shortCode
	count, err := c.redisClient.Client.Exists(ctx, key).Result()
	if err != nil {
		c.metrics.RecordCacheError()
		return false, err
	}
	return count > 0, nil
}
