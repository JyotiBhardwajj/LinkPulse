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
	healthHandler *handler.HealthHandler,
	linkHandler *handler.LinkHandler,
	userHandler *handler.UserHandler,
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

	// API Group
	api := r.Group("/api/v1")
	{
		// Authentication (Placeholder)
		api.POST("/users/register", userHandler.Register)

		links := api.Group("/links")
		{
			links.POST("", linkHandler.Shorten)
			links.GET("/:code/stats", middleware.Auth(), linkHandler.GetStats)
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
