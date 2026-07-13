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

// Create creates a new shortened link.
//
// @Summary      Create short link
// @Description  Shortens a long URL and returns the short link details.
// @Tags         links
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      models.CreateLinkRequest  true  "Link creation payload"
// @Success      201      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      422      {object}  models.ValidationErrorResponse
// @Router       /api/v1/links [post]
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

// Resolve resolves a short code and redirects to the original URL.
//
// @Summary      Resolve short link
// @Description  Redirects to the original URL for the given short code. Click analytics are recorded asynchronously.
// @Tags         redirect
// @Param        code  path  string  true  "Short code"
// @Success      302
// @Failure      404  {object}  models.ErrorResponse
// @Failure      410  {object}  models.ErrorResponse
// @Router       /r/{code} [get]
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

// Get retrieves a specific link by ID.
//
// @Summary      Get link by ID
// @Description  Returns details of a specific shortened link owned by the authenticated user.
// @Tags         links
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Link UUID"
// @Success      200  {object}  models.SuccessResponse
// @Success      304  "Not Modified"
// @Failure      401  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /api/v1/links/{id} [get]
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

// List returns all links owned by the authenticated user.
//
// @Summary      List links
// @Description  Returns a paginated list of all shortened links created by the authenticated user.
// @Tags         links
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        limit     query     int     false  "Items per page (default 20, max 100)"
// @Param        search    query     string  false  "Search by URL or title"
// @Param        sort      query     string  false  "Sort field: created_at, updated_at, expires_at"
// @Param        order     query     string  false  "Sort order: asc, desc"
// @Param        status    query     string  false  "Filter by status: active, expired, inactive, deleted"
// @Success      200       {object}  models.PaginatedResponse
// @Success      304       "Not Modified"
// @Failure      401       {object}  models.ErrorResponse
// @Router       /api/v1/links [get]
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

// Update partially updates a link's configuration.
//
// @Summary      Update link
// @Description  Updates the title, expiry date, or active status of an existing link.
// @Tags         links
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                   true  "Link UUID"
// @Param        request  body      models.UpdateLinkRequest  true  "Update payload"
// @Success      200      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      404      {object}  models.ErrorResponse
// @Failure      422      {object}  models.ValidationErrorResponse
// @Router       /api/v1/links/{id} [patch]
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

// Delete soft-deletes a link.
//
// @Summary      Delete link
// @Description  Soft-deletes a shortened link. The short code will no longer resolve.
// @Tags         links
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Link UUID"
// @Success      204  "No Content"
// @Failure      401  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /api/v1/links/{id} [delete]
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

// GetStats returns click statistics for a specific link.
//
// @Summary      Get link statistics
// @Description  Returns total click count and basic metrics for a specific shortened link.
// @Tags         links
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Link UUID"
// @Success      200  {object}  models.SuccessResponse
// @Failure      401  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /api/v1/links/{id}/stats [get]
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
