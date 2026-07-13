// Package routes handles route registrations and middleware piping.
package routes

import (
	"net/http"
	"time"

	"linkpulse/internal/handler"
	"linkpulse/internal/metrics"
	"linkpulse/internal/middleware"
	"linkpulse/internal/models"

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

	// Apply Middlewares in correct order
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	if metricsTracker != nil {
		r.Use(middleware.MetricsMiddleware(metricsTracker))
	}
	r.Use(middleware.Logger())
	r.Use(middleware.Timeout(timeoutDuration))
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit())

	// Diagnostic Endpoints
	r.GET("/health", healthHandler.Check)
	r.GET("/ready", healthHandler.CheckReady)

	if metricsTracker != nil {
		if exposer, ok := metricsTracker.(interface{ HTTPHandler() http.Handler }); ok {
			r.GET("/metrics", gin.WrapH(exposer.HTTPHandler()))
		}
	}

	// Redirect Endpoint (Optimized path)
	r.GET("/r/:code", linkHandler.Resolve)

	// Static Swagger spec file
	r.StaticFile("/docs/swagger.json", "./docs/swagger.json")

	// Instantiate authorization middleware wrapper
	authMiddleware := middleware.Auth(jwtSecret, jwtIssuer)

	// API Group V1
	v1 := r.Group("/api/v1")
	registerV1Routes(v1, authMiddleware, healthHandler, linkHandler, userHandler, authHandler, analyticsHandler)

	// API Group V2 (Future evolution placeholder)
	v2 := r.Group("/api/v2")
	registerV2Routes(v2)

	// Swagger Placeholder API endpoint
	r.GET("/swagger/*any", func(c *gin.Context) {
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
		analytics.GET("/top-links", analyticsHandler.GetTopLinks)
		analytics.GET("/devices", analyticsHandler.GetDeviceDistribution)
		analytics.GET("/browsers", analyticsHandler.GetBrowserDistribution)
		analytics.GET("/referrers", analyticsHandler.GetReferrerDistribution)
	}

	// Admin Routes Group (RBAC Protected placeholder)
	admin := api.Group("/admin", authMiddleware, middleware.RequireRole(models.RoleAdmin))
	{
		admin.Any("/*path", func(c *gin.Context) {
			c.Status(http.StatusNotImplemented)
		})
	}
}

// registerV2Routes registers version 2 placeholder.
func registerV2Routes(api *gin.RouterGroup) {
	// Future evolution endpoints go here
}
