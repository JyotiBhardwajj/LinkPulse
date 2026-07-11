// Package handler implements HTTP controllers and request parsers.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"
	"linkpulse/internal/service"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// LinkHandler controllers HTTP operations on links.
type LinkHandler struct {
	linkService service.LinkService
}

// NewLinkHandler creates a new LinkHandler instance.
func NewLinkHandler(linkService service.LinkService) *LinkHandler {
	return &LinkHandler{linkService: linkService}
}

// Shorten binds the request body and invokes the service to generate a shortened URL.
func (h *LinkHandler) Shorten(c *gin.Context) {
	var req models.ShortenLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_REQUEST_BODY")
		return
	}

	// For Day 1, auth is placeholder so we create guest links (nil user ID)
	resp, err := h.linkService.Shorten(c.Request.Context(), req, nil)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "SHORTEN_FAILED")
		return
	}

	// Construct the absolute shortened URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	resp.ShortURL = scheme + "://" + c.Request.Host + "/r/" + resp.ShortCode

	utils.SendSuccess(c, http.StatusCreated, "Link shortened successfully", resp)
}

// Resolve looks up the short code and redirects the client, writing click analytics asynchronously.
func (h *LinkHandler) Resolve(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		utils.SendError(c, http.StatusBadRequest, "short code is required", "MISSING_SHORT_CODE")
		return
	}

	// Resolve the original URL (cache-first check)
	originalURL, err := h.linkService.Resolve(c.Request.Context(), code)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "RESOLVE_FAILED")
		return
	}

	// Extract click details for analytics
	userAgent := c.Request.UserAgent()
	referrer := c.GetHeader("Referer")
	clientIP := c.ClientIP()
	ipHash := utils.HashIP(clientIP)

	clickDetails := models.ClickDetails{
		IPHash:    ipHash,
		UserAgent: userAgent,
		Referrer:  referrer,
		// Analytics parsing placeholders (real parsing would use useragent/geoip libs)
		Country: "Unknown",
		City:    "Unknown",
		Browser: "Unknown",
		OS:      "Unknown",
		Device:  "Unknown",
	}

	// Redirect client immediately to minimize latency
	c.Redirect(http.StatusFound, originalURL)

	// Fire-and-forget click analytics tracking to keep redirect fast and enable future asynchronous scale
	go func(shortCode string, details models.ClickDetails) {
		// Use background context as the original request context will be cancelled after redirect response completes
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.linkService.RecordClick(ctx, shortCode, details); err != nil {
			slog.Error("Asynchronous click tracking failed", "code", shortCode, "error", err)
		}
	}(code, clickDetails)
}

// GetStats returns usage statistics for a specific shortened link.
func (h *LinkHandler) GetStats(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		utils.SendError(c, http.StatusBadRequest, "short code is required", "MISSING_SHORT_CODE")
		return
	}

	// For Day 1, since authentication is placeholder, we use a zero UUID as stub or return unauthorized
	// Once JWT middleware is active, we would read the user ID from the context.
	status := domainErrors.MapToHTTPStatus(domainErrors.ErrUnauthorized)
	utils.SendError(c, status, "Authentication required to view link stats", "UNAUTHORIZED")
}
