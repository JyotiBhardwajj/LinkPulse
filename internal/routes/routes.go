// Package routes handles route registrations and middleware piping.
package routes

import (
	"net/http"
	"time"

	"linkpulse/internal/handler"
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
) *gin.Engine {
	// Disable Gin default logging to use our structured slog middleware
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Apply Middlewares in correct order
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.Timeout(timeoutDuration))
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit())

	// Diagnostic Endpoints
	r.GET("/health", healthHandler.Check)

	// Redirect Endpoint (Optimized path)
	r.GET("/r/:code", linkHandler.Resolve)

	// Instantiate authorization middleware wrapper
	authMiddleware := middleware.Auth(jwtSecret, jwtIssuer)

	// API Group
	api := r.Group("/api/v1")
	{
		// Authentication Routes Group
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authMiddleware, authHandler.Logout)
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
			links.GET("/:code/stats", linkHandler.GetStats)
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
	}

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
