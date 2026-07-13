// Package routes handles route registrations and middleware piping.
package routes

import (
	"net/http"
	"time"

	"linkpulse/internal/handler"
	"linkpulse/internal/metrics"
	"linkpulse/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter binds HTTP endpoints to handlers and registers middlewares.
func SetupRouter(
	timeoutDuration time.Duration,
	jwtSecret string,
	jwtIssuer string,
	healthHandler *handler.HealthHandler,
	linkHandler *handler.LinkHandler,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	analyticsHandler *handler.AnalyticsHandler,
	metricsTracker metrics.Metrics,
) *gin.Engine {
	// Disable Gin default logging to use our structured slog middleware
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Apply globally safe Middlewares
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// Diagnostic Endpoints (no logging, timeout, rate limiting, or metrics)
	healthGroup := r.Group("/health")
	healthGroup.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})
	{
		healthGroup.GET("/live", healthHandler.Live)
		healthGroup.GET("/ready", healthHandler.Ready)
		healthGroup.GET("/startup", healthHandler.Startup)
	}

	// For backward compatibility:
	r.GET("/health", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		healthHandler.Live(c)
	})
	r.GET("/ready", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		healthHandler.Ready(c)
	})

	// Create a sub-group for all other application routes that require standard middlewares
	mainGroup := r.Group("/")
	if metricsTracker != nil {
		mainGroup.Use(middleware.MetricsMiddleware(metricsTracker))
	}
	mainGroup.Use(middleware.Logger())
	mainGroup.Use(middleware.Timeout(timeoutDuration))
	mainGroup.Use(middleware.RateLimit())

	if metricsTracker != nil {
		if exposer, ok := metricsTracker.(interface{ HTTPHandler() http.Handler }); ok {
			mainGroup.GET("/metrics", gin.WrapH(exposer.HTTPHandler()))
		}
	}

	// Redirect Endpoint (Optimized path)
	mainGroup.GET("/r/:code", linkHandler.Resolve)

	// Static Swagger spec file
	mainGroup.StaticFile("/docs/swagger.json", "./docs/swagger.json")

	// Instantiate authorization middleware wrapper
	authMiddleware := middleware.Auth(jwtSecret, jwtIssuer)

	// API Group V1
	v1 := mainGroup.Group("/api/v1")
	registerV1Routes(v1, authMiddleware, healthHandler, linkHandler, userHandler, authHandler, analyticsHandler)

	// API Group V2 (Future evolution placeholder)
	v2 := mainGroup.Group("/api/v2")
	registerV2Routes(v2)

	// Swagger Placeholder API endpoint
	mainGroup.GET("/swagger/*any", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":     "Swagger UI Placeholder. Run 'make swagger' to compile and generate Swagger docs.",
			"doc_version": "1.0.0",
			"spec_url":    "/docs/swagger.json",
		})
	})

	return r
}

// registerV1Routes registers version 1 endpoints.
func registerV1Routes(
	api *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
	healthHandler *handler.HealthHandler,
	linkHandler *handler.LinkHandler,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	analyticsHandler *handler.AnalyticsHandler,
) {
	// Authentication Routes Group
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authMiddleware, authHandler.Logout)
		auth.GET("/sessions", authMiddleware, authHandler.GetSessions)
		auth.POST("/logout-all", authMiddleware, authHandler.LogoutAll)
	}

	// User Profile Routes Group
	users := api.Group("/users")
	{
		// Protected user current profile lookup
		users.GET("/me", authMiddleware, userHandler.Me)
	}

	// Shortened Links Routes Group (Fully authenticated for CRUD operations)
	links := api.Group("/links", authMiddleware)
	{
		links.POST("", linkHandler.Create)
		links.GET("", linkHandler.List)
		links.GET("/:id", linkHandler.Get)
		links.PATCH("/:id", linkHandler.Update)
		links.DELETE("/:id", linkHandler.Delete)
		links.GET("/:id/stats", linkHandler.GetStats)
		links.GET("/:id/analytics", analyticsHandler.GetLinkAnalytics)
	}

	// Analytics Routes Group (Fully authenticated for metrics dashboard)
	analytics := api.Group("/analytics", authMiddleware)
	{
		analytics.GET("/overview", analyticsHandler.GetOverview)
		analytics.GET("/clicks", analyticsHandler.GetClicksOverTime)
	}
}

// registerV2Routes registers version 2 placeholder endpoints.
func registerV2Routes(api *gin.RouterGroup) {
	// Placeholder for future endpoints
}
