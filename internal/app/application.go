// Package app orchestrates the lifecycle and configurations bootstrap.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"linkpulse/internal/cache"
	"linkpulse/internal/config"
	"linkpulse/internal/database"
	"linkpulse/internal/handler"
	"linkpulse/internal/logger"
	"linkpulse/internal/metrics"
	"linkpulse/internal/repository"
	"linkpulse/internal/routes"
	"linkpulse/internal/service"
	"linkpulse/internal/worker"
)

// Application manages boot tasks, resource connections, and graceful shutdowns.
type Application struct {
	config           *config.Config
	db               *database.PostgresDB
	redis            *cache.RedisClient
	httpServer       *http.Server
	workerPool       worker.WorkerPool
	cleanupScheduler *worker.CleanupScheduler
	auditLogger      logger.AsyncAuditLogger
}

// NewApplication instantiates all components and connects to external stores.
func NewApplication() (*Application, error) {
	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	// 2. Initialize structured logging
	_ = logger.InitLogger(cfg.Server.Env, cfg.LogLevel)
	slog.Info("Initializing LinkPulse application components")

	// 3. Initialize and start Asynchronous Audit Logger
	auditLogger := logger.InitAuditLogger(1000)
	auditLogger.Start(context.Background())

	// 4. Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		_ = auditLogger.Close(context.Background())
		return nil, fmt.Errorf("postgres error: %w", err)
	}

	// 5. Verify PostgreSQL migrations are applied
	tables := []string{"users", "links", "analytics", "refresh_tokens"}
	for _, table := range tables {
		if !db.DB.Migrator().HasTable(table) {
			_ = db.Close()
			_ = auditLogger.Close(context.Background())
			return nil, fmt.Errorf("migration verification failed: table '%s' is missing", table)
		}
	}

	// 6. Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		_ = db.Close()
		_ = auditLogger.Close(context.Background())
		return nil, fmt.Errorf("redis error: %w", err)
	}

	// Initialize metrics
	var metricsTracker metrics.Metrics
	if cfg.Metrics.EnableMetrics {
		var registry interface{}
		metricsTracker, registry = metrics.GetProductionMetrics(cfg.Metrics.MetricsNamespace, cfg.Metrics.MetricsSubsystem)
		if registry == nil {
			slog.Warn("Failed to initialize production metrics registry, falling back to NoOp")
			metricsTracker = metrics.NewNoOpMetrics()
		} else {
			slog.Info("Successfully initialized Prometheus metrics")
			// Register GORM metrics plugin
			err = db.DB.Use(database.NewMetricsPlugin(metricsTracker))
			if err != nil {
				slog.Error("Failed to register GORM metrics plugin", "error", err)
			} else {
				slog.Info("Successfully registered GORM metrics plugin")
			}
		}
	} else {
		metricsTracker = metrics.NewNoOpMetrics()
	}

	// 7. Initialize LinkCache (using prefix namespacing)
	linkCache := cache.NewLinkCache(redisClient, cfg.Cache.Prefix, metricsTracker)

	// 8. Initialize Repositories (RepositoryManager) and TransactionManager
	repoMgr := repository.NewRepositoryManager(db.DB)
	txMgr := repository.NewTransactionManager(db.DB)

	// 9. Initialize WorkerPool
	workerPool := worker.NewWorkerPool(repoMgr.Analytics(), cfg.Worker.Count, cfg.Worker.QueueSize, metricsTracker)

	// 10. Initialize Services
	userService := service.NewUserService(repoMgr.Users())
	linkService := service.NewLinkService(
		repoMgr.Links(),
		repoMgr.Analytics(),
		linkCache,
		cfg.Server.ShortCodeLength,
		cfg.Server.MaxGenerationRetries,
		cfg.Server.BaseURL,
		cfg.Cache.TTL,
		metricsTracker,
	)
	authService := service.NewAuthService(
		repoMgr.Users(),
		repoMgr.RefreshTokens(),
		txMgr,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
		cfg.JWT.Issuer,
		cfg.JWT.MaxSessionsPerUser,
		metricsTracker,
	)
	analyticsService := service.NewAnalyticsService(repoMgr.Analytics(), repoMgr.Links(), metricsTracker)

	// Initialize ReadinessService with parallel checker map
	checkers := map[string]service.ReadinessChecker{
		"database":    db,
		"redis":       redisClient,
		"worker_pool": workerPool,
	}
	readyService := service.NewReadinessService(checkers)

	// 11. Initialize CleanupScheduler
	cleanupScheduler := worker.NewCleanupScheduler(linkService, cfg.Cleanup.Interval)

	// 12. Initialize Handlers
	healthHandler := handler.NewHealthHandler(readyService, cfg.Server.Version, cfg.Server.GitCommit, cfg.Server.BuildTime, cfg.Server.Env)
	linkHandler := handler.NewLinkHandler(linkService, workerPool)
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)

	// 13. Setup HTTP Router
	router := routes.SetupRouter(cfg.Server.RequestTimeout, cfg.JWT.Secret, cfg.JWT.Issuer, healthHandler, linkHandler, userHandler, authHandler, analyticsHandler, metricsTracker)

	// 14. Instantiate HTTP server wrapper
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Application{
		config:           cfg,
		db:               db,
		redis:            redisClient,
		httpServer:       server,
		workerPool:       workerPool,
		cleanupScheduler: cleanupScheduler,
		auditLogger:      auditLogger,
	}, nil
}

// Run starts the HTTP server in a goroutine and handles termination signals for graceful shutdown.
func (a *Application) Run() error {
	slog.Info("Starting HTTP server", "address", a.httpServer.Addr, "env", a.config.Server.Env)

	// Start WorkerPool background channels
	a.workerPool.Start(context.Background())

	// Start Background Cleanup Scheduler ticker loop
	a.cleanupScheduler.Start(context.Background())

	// Start server in background goroutine
	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("Termination signal received. Starting graceful shutdown sequence", "signal", sig.String())

	// Shutdown timeout context (10 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Stop accepting HTTP requests
	if err := a.httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown encountered an error", "error", err)
	} else {
		slog.Info("HTTP server stopped accepting new requests")
	}

	// 2. Close Async Audit Logger (drains all remaining logs)
	if err := a.auditLogger.Close(ctx); err != nil {
		slog.Error("Error closing audit logger gracefully", "error", err)
	} else {
		slog.Info("Audit logger closed successfully")
	}

	// Stop background cleanup ticker
	a.cleanupScheduler.Stop()

	// 3. Drain Worker Pool (waits for queued events to flush to PostgreSQL)
	if err := a.workerPool.Shutdown(ctx); err != nil {
		slog.Error("Worker pool shutdown encountered an error", "error", err)
	} else {
		slog.Info("Worker pool gracefully drained and stopped")
	}

	// 4. Close Redis connections
	if err := a.redis.Close(); err != nil {
		slog.Error("Error closing Redis client connection", "error", err)
	} else {
		slog.Info("Redis connection client closed successfully")
	}

	// 5. Close PostgreSQL connections (after worker pool has completed GORM inserts)
	if err := a.db.Close(); err != nil {
		slog.Error("Error closing PostgreSQL connection pool", "error", err)
	} else {
		slog.Info("PostgreSQL connection pool closed successfully")
	}

	// 6. Flush Logger
	slog.Info("Graceful shutdown completed. Exiting safely.")
	return nil
}
