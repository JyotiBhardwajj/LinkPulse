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
)

// Application coordinates database connections, dependency injection, and runtime state.
type Application struct {
	config     *config.Config
	db         *database.PostgresDB
	redis      *cache.RedisClient
	httpServer *http.Server
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
		// Clean up Postgres connection if Redis fail
		_ = db.Close()
		return nil, fmt.Errorf("redis error: %w", err)
	}

	// 5. Initialize LinkCache
	linkCache := cache.NewLinkCache(redisClient, cfg.Cache.TTL)

	// 6. Initialize Repositories (RepositoryManager)
	repoMgr := repository.NewRepositoryManager(db.DB)

	// 7. Initialize Services
	userService := service.NewUserService(repoMgr.Users())
	linkService := service.NewLinkService(repoMgr.Links(), repoMgr.Analytics(), linkCache, cfg.Server.ShortCodeLength, cfg.Server.MaxGenerationRetries)

	// 8. Initialize Handlers
	healthHandler := handler.NewHealthHandler(db, redisClient, cfg.BuildVersion, cfg.GitCommit, cfg.Server.Env)
	linkHandler := handler.NewLinkHandler(linkService)
	userHandler := handler.NewUserHandler(userService)

	// 9. Setup HTTP Router
	router := routes.SetupRouter(cfg.Server.RequestTimeout, healthHandler, linkHandler, userHandler)

	// 10. Instantiate HTTP server wrapper
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Application{
		config:     cfg,
		db:         db,
		redis:      redisClient,
		httpServer: server,
	}, nil
}

// Run starts the HTTP server in a goroutine and handles termination signals for graceful shutdown.
func (a *Application) Run() error {
	slog.Info("Starting HTTP server", "address", a.httpServer.Addr, "env", a.config.Server.Env)

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

	// 1. Close HTTP Server (stops accepting new connections, waits for active requests)
	if err := a.httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown encountered an error", "error", err)
	} else {
		slog.Info("HTTP server stopped accepting new requests")
	}

	// 2. Close PostgreSQL connections
	if err := a.db.Close(); err != nil {
		slog.Error("Error closing PostgreSQL connection pool", "error", err)
	} else {
		slog.Info("PostgreSQL connection pool closed successfully")
	}

	// 3. Close Redis connections
	if err := a.redis.Close(); err != nil {
		slog.Error("Error closing Redis client connection", "error", err)
	} else {
		slog.Info("Redis connection client closed successfully")
	}

	slog.Info("Graceful shutdown completed. Exiting safely.")
	return nil
}
