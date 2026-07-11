// Package cache manages Redis client setup and application-specific cache layers.
package cache

import (
	"context"
	"time"
)

// LinkCache defines the caching operations for resolved URLs.
type LinkCache interface {
	// Get retrieves the original URL associated with the short code.
	Get(ctx context.Context, code string) (string, error)

	// Set stores the original URL associated with the short code using the configured TTL.
	Set(ctx context.Context, code string, originalURL string) error

	// Delete removes the cached short code resolution.
	Delete(ctx context.Context, code string) error
}

type linkCache struct {
	redisClient *RedisClient
	ttl         time.Duration
}

// NewLinkCache instantiates a LinkCache implementation backed by Redis.
func NewLinkCache(redisClient *RedisClient, ttl time.Duration) LinkCache {
	return &linkCache{
		redisClient: redisClient,
		ttl:         ttl,
	}
}

// Get checks the Redis cache for the given short code.
func (c *linkCache) Get(ctx context.Context, code string) (string, error) {
	val, err := c.redisClient.Client.Get(ctx, code).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// Set saves the short code to original URL mapping in Redis with the specified TTL.
func (c *linkCache) Set(ctx context.Context, code string, originalURL string) error {
	return c.redisClient.Client.Set(ctx, code, originalURL, c.ttl).Err()
}

// Delete removes the cached mapping from Redis.
func (c *linkCache) Delete(ctx context.Context, code string) error {
	return c.redisClient.Client.Del(ctx, code).Err()
}
