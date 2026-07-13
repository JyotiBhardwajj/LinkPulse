// Package handler implements HTTP controllers and request parsers.
package handler

import (
	"net/http"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/middleware"
	"linkpulse/internal/models"
	"linkpulse/internal/service"
	"linkpulse/internal/utils"
	"linkpulse/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LinkHandler controllers HTTP operations on links.
type LinkHandler struct {
	linkService service.LinkService
	workerPool  worker.WorkerPool
}

// NewLinkHandler creates a new LinkHandler instance.
func NewLinkHandler(linkService service.LinkService, workerPool worker.WorkerPool) *LinkHandler {
	return &LinkHandler{
		linkService: linkService,
		workerPool:  workerPool,
	}
}

// Create binds request payload and creates a shortened mapping.
func (h *LinkHandler) Create(c *gin.Context) {
	var req models.CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	resp, err := h.linkService.Create(c.Request.Context(), req, &authCtx.UserID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_CREATION_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusCreated, "Link shortened successfully", resp)
}

// Resolve processes short code lookup and issues 302 HTTP redirection.
// Offloads click metrics tracking asynchronously using the background worker pool.
func (h *LinkHandler) Resolve(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		utils.SendError(c, http.StatusBadRequest, "short code parameter is required", "MISSING_SHORT_CODE")
		return
	}

	cached, err := h.linkService.Resolve(c.Request.Context(), code)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "RESOLVE_FAILED")
		return
	}

	// HTTP redirect (302 Found) executed strictly inside HTTP handler layer
	c.Redirect(http.StatusFound, cached.OriginalURL)

	// Asynchronous metrics analytics logging offloaded to WorkerPool queue
	clientIP := c.ClientIP()
	ipHash := utils.HashIP(clientIP)
	userAgent := c.Request.UserAgent()
	referrer := c.GetHeader("Referer")

	event := worker.ClickEvent{
		LinkID:        cached.ID,
		Timestamp:     time.Now(),
		UserAgent:     userAgent,
		Referrer:      referrer,
		IPAddressHash: ipHash,
	}

	// Submit asynchronously. Never blocks redirect execution thread.
	_ = h.workerPool.Submit(c.Request.Context(), event)
}

// Get retrieves details of a specific link owned by the user.
func (h *LinkHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "id",
				Rule:    "uuid",
				Message: "The id field must be a valid UUID",
			},
		})
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	resp, err := h.linkService.GetByID(c.Request.Context(), id, authCtx.UserID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_NOT_FOUND")
		return
	}

	// Deterministic ETag-based caching support
	etag := utils.ComputeETag(resp)
	c.Header("Cache-Control", "public, max-age=0, must-revalidate")
	c.Header("ETag", etag)
	if utils.CheckIfNoneMatch(c, etag) {
		c.Status(http.StatusNotModified)
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Link retrieved successfully", resp)
}

// List returns all links registered by the authenticated user.
func (h *LinkHandler) List(c *gin.Context) {
	var q models.ListLinksQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, q))
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	resp, err := h.linkService.List(c.Request.Context(), authCtx.UserID, q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_LIST_FAILED")
		return
	}

	// Deterministic ETag-based caching support for listing resource
	etag := utils.ComputeETag(resp)
	c.Header("Cache-Control", "public, max-age=0, must-revalidate")
	c.Header("ETag", etag)
	if utils.CheckIfNoneMatch(c, etag) {
		c.Status(http.StatusNotModified)
		return
	}

	utils.SendPaginatedSuccess(c, http.StatusOK, "Links retrieved successfully", resp, resp.Page, resp.Limit, resp.Total, resp.TotalPages)
}

// Update partial-modifies link configurations.
func (h *LinkHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "id",
				Rule:    "uuid",
				Message: "The id field must be a valid UUID",
			},
		})
		return
	}

	var req models.UpdateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	resp, err := h.linkService.Update(c.Request.Context(), id, authCtx.UserID, req)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_UPDATE_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Link updated successfully", resp)
}

// Delete soft-removes the resource mapping.
func (h *LinkHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "id",
				Rule:    "uuid",
				Message: "The id field must be a valid UUID",
			},
		})
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	err = h.linkService.Delete(c.Request.Context(), id, authCtx.UserID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_DELETE_FAILED")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetStats returns click analytics counts for a specific shortened link.
func (h *LinkHandler) GetStats(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		code = c.Param("id")
	}
	if code == "" {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "code",
				Rule:    "required",
				Message: "The code parameter is required",
			},
		})
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	resp, err := h.linkService.GetStats(c.Request.Context(), code, authCtx.UserID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_STATS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Link statistics retrieved successfully", resp)
}
