package health

import (
	"context"
	"errors"

	"linkpulse/internal/cache"
	"linkpulse/internal/config"
	"linkpulse/internal/database"
	"linkpulse/internal/metrics"
	"linkpulse/internal/worker"
)

// PostgresChecker validates database connectivity.
type PostgresChecker struct {
	pg *database.PostgresDB
}

func NewPostgresChecker(pg *database.PostgresDB) Checker {
	return &PostgresChecker{pg: pg}
}

func (c *PostgresChecker) Name() string {
	return "postgres"
}

func (c *PostgresChecker) Check(ctx context.Context) error {
	if c.pg == nil {
		return errors.New("postgres database instance is nil")
	}
	return c.pg.Ping(ctx)
}

func (c *PostgresChecker) IsCritical() bool {
	return true
}

// RedisChecker validates cache connectivity.
type RedisChecker struct {
	client *cache.RedisClient
}

func NewRedisChecker(client *cache.RedisClient) Checker {
	return &RedisChecker{client: client}
}

func (c *RedisChecker) Name() string {
	return "redis"
}

func (c *RedisChecker) Check(ctx context.Context) error {
	if c.client == nil {
		return errors.New("redis client instance is nil")
	}
	return c.client.Ping(ctx)
}

func (c *RedisChecker) IsCritical() bool {
	return false // Redis is optional
}

// WorkerPoolChecker validates background worker pool operation.
type WorkerPoolChecker struct {
	pool worker.WorkerPool
}

func NewWorkerPoolChecker(pool worker.WorkerPool) Checker {
	return &WorkerPoolChecker{pool: pool}
}

func (c *WorkerPoolChecker) Name() string {
	return "worker_pool"
}

func (c *WorkerPoolChecker) Check(ctx context.Context) error {
	if c.pool == nil {
		return errors.New("worker pool instance is nil")
	}
	return c.pool.Ready(ctx)
}

func (c *WorkerPoolChecker) IsCritical() bool {
	return true
}

// MetricsChecker validates metrics tracking initialization.
type MetricsChecker struct {
	tracker metrics.Metrics
}

func NewMetricsChecker(tracker metrics.Metrics) Checker {
	return &MetricsChecker{tracker: tracker}
}

func (c *MetricsChecker) Name() string {
	return "metrics"
}

func (c *MetricsChecker) Check(ctx context.Context) error {
	if c.tracker == nil {
		return errors.New("metrics subsystem is not initialized")
	}
	return nil
}

func (c *MetricsChecker) IsCritical() bool {
	return false // Metrics are optional
}

// ConfigChecker validates configuration settings presence.
type ConfigChecker struct {
	cfg *config.Config
}

func NewConfigChecker(cfg *config.Config) Checker {
	return &ConfigChecker{cfg: cfg}
}

func (c *ConfigChecker) Name() string {
	return "config"
}

func (c *ConfigChecker) Check(ctx context.Context) error {
	if c.cfg == nil {
		return errors.New("configuration is nil")
	}
	if c.cfg.Server.Port == "" {
		return errors.New("required configuration setting (Server Port) is missing")
	}
	return nil
}

func (c *ConfigChecker) IsCritical() bool {
	return true
}
