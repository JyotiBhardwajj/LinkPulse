// Package app contains the application bootsrapping and startup sequence logic.
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
	"linkpulse/internal/repository"
	"linkpulse/internal/routes"
	"linkpulse/internal/service"
	"linkpulse/internal/worker"
)

// Application coordinates database connections, dependency injection, and runtime state.
type Application struct {
	config           *config.Config
	db               *database.PostgresDB
	redis            *cache.RedisClient
	httpServer       *http.Server
	workerPool       worker.WorkerPool
	cleanupScheduler *worker.CleanupScheduler
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

	// 3. Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("postgres error: %w", err)
	}

	// 4. Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		// Clean up Postgres connection if Redis fails
		_ = db.Close()
		return nil, fmt.Errorf("redis error: %w", err)
	}

	// 5. Initialize LinkCache (using prefix namespacing)
	linkCache := cache.NewLinkCache(redisClient, cfg.Cache.Prefix)

	// 6. Initialize Repositories (RepositoryManager)
	repoMgr := repository.NewRepositoryManager(db.DB)

	// 7. Initialize WorkerPool
	workerPool := worker.NewWorkerPool(repoMgr.Analytics(), cfg.Worker.Count, cfg.Worker.QueueSize)

	// 8. Initialize Services
	userService := service.NewUserService(repoMgr.Users())
	linkService := service.NewLinkService(
		repoMgr.Links(),
		repoMgr.Analytics(),
		linkCache,
		cfg.Server.ShortCodeLength,
		cfg.Server.MaxGenerationRetries,
		cfg.Server.BaseURL,
		cfg.Cache.TTL,
	)
	authService := service.NewAuthService(repoMgr.Users(), repoMgr.RefreshTokens(), cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL, cfg.JWT.Issuer)
	analyticsService := service.NewAnalyticsService(repoMgr.Analytics(), repoMgr.Links())

	// 9. Initialize CleanupScheduler
	cleanupScheduler := worker.NewCleanupScheduler(linkService, cfg.Cleanup.Interval)

	// 10. Initialize Handlers
	healthHandler := handler.NewHealthHandler(db, redisClient, cfg.BuildVersion, cfg.GitCommit, cfg.Server.Env)
	linkHandler := handler.NewLinkHandler(linkService, workerPool)
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)

	// 11. Setup HTTP Router
	router := routes.SetupRouter(cfg.Server.RequestTimeout, cfg.JWT.Secret, cfg.JWT.Issuer, healthHandler, linkHandler, userHandler, authHandler, analyticsHandler)

	// 12. Instantiate HTTP server wrapper
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

	// 1. Stop background cleanup scheduler ticker
	a.cleanupScheduler.Stop()

	// 2. Close HTTP Server (stops accepting new connections, waits for active requests)
	if err := a.httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown encountered an error", "error", err)
	} else {
		slog.Info("HTTP server stopped accepting new requests")
	}

	// 3. Stop worker pool and wait for remaining events
	if err := a.workerPool.Shutdown(ctx); err != nil {
		slog.Error("Worker pool shutdown encountered an error", "error", err)
	} else {
		slog.Info("Worker pool gracefully stopped")
	}

	// 4. Close PostgreSQL connections
	if err := a.db.Close(); err != nil {
		slog.Error("Error closing PostgreSQL connection pool", "error", err)
	} else {
		slog.Info("PostgreSQL connection pool closed successfully")
	}

	// 5. Close Redis connections
	if err := a.redis.Close(); err != nil {
		slog.Error("Error closing Redis client connection", "error", err)
	} else {
		slog.Info("Redis connection client closed successfully")
	}

	slog.Info("Graceful shutdown completed. Exiting safely.")
	return nil
}
