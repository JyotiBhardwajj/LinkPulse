// Package cache manages Redis client setup and application-specific cache layers.
package cache

import (
	"context"
	"fmt"
	"log/slog"

	"linkpulse/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the go-redis client.
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates and validates a new Redis connection.
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Verify connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	slog.Info("Successfully connected to Redis cache")

	return &RedisClient{
		Client: client,
	}, nil
}

// Ping checks if Redis is still reachable.
func (r *RedisClient) Ping(ctx context.Context) error {
	if r.Client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return r.Client.Ping(ctx).Err()
}

// Ready satisfies the ReadinessChecker interface.
func (r *RedisClient) Ready(ctx context.Context) error {
	return r.Ping(ctx)
}

// Close gracefully closes the Redis client connections.
func (r *RedisClient) Close() error {
	if r.Client == nil {
		return nil
	}
	slog.Info("Closing Redis connection client")
	return r.Client.Close()
}
