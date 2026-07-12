package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/middleware"
	"linkpulse/internal/models"
	"linkpulse/internal/service"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AnalyticsHandler controllers HTTP requests for compiles analytics.
type AnalyticsHandler struct {
	analyticsService service.AnalyticsService
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(analyticsService service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

// GetOverview returns total and active system statistics for the user's links.
func (h *AnalyticsHandler) GetOverview(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	q := models.AnalyticsQuery{
		UserID: authCtx.UserID,
	}

	overview, err := h.analyticsService.GetOverview(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Analytics overview compiled successfully", overview)
}

// GetClicksOverTime returns time-series click buckets.
func (h *AnalyticsHandler) GetClicksOverTime(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	start, end, err := h.parseDates(c)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_DATE_FORMAT")
		return
	}

	interval := c.DefaultQuery("interval", "day")

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		StartDate: start,
		EndDate:   end,
		Interval:  interval,
	}

	points, err := h.analyticsService.GetClicksOverTime(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Time series compiled successfully", points)
}

// GetTopLinks returns top links by click counts.
func (h *AnalyticsHandler) GetTopLinks(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "limit parameter must be an integer", "INVALID_LIMIT")
		return
	}

	q := models.AnalyticsQuery{
		UserID: authCtx.UserID,
		Limit:  limit,
	}

	topLinks, err := h.analyticsService.GetTopLinks(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Top performing links retrieved successfully", topLinks)
}

// GetDeviceDistribution returns clicks percentages grouped by device platforms.
func (h *AnalyticsHandler) GetDeviceDistribution(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	start, end, err := h.parseDates(c)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_DATE_FORMAT")
		return
	}

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		StartDate: start,
		EndDate:   end,
	}

	devices, err := h.analyticsService.GetDeviceDistribution(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Device distribution compiled successfully", devices)
}

// GetBrowserDistribution returns clicks percentages grouped by browser agents.
func (h *AnalyticsHandler) GetBrowserDistribution(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	start, end, err := h.parseDates(c)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_DATE_FORMAT")
		return
	}

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		StartDate: start,
		EndDate:   end,
	}

	browsers, err := h.analyticsService.GetBrowserDistribution(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Browser distribution compiled successfully", browsers)
}

// GetReferrerDistribution returns top domain referrers.
func (h *AnalyticsHandler) GetReferrerDistribution(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	start, end, err := h.parseDates(c)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_DATE_FORMAT")
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "limit parameter must be an integer", "INVALID_LIMIT")
		return
	}

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		StartDate: start,
		EndDate:   end,
		Limit:     limit,
	}

	referrers, err := h.analyticsService.GetReferrerDistribution(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Referrers distribution compiled successfully", referrers)
}

// GetLinkAnalytics returns complete metrics for a single link ID (with ownership checks).
func (h *AnalyticsHandler) GetLinkAnalytics(c *gin.Context) {
	idStr := c.Param("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid link ID format", "INVALID_LINK_ID")
		return
	}

	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "user authentication context required", "UNAUTHORIZED")
		return
	}

	start, end, err := h.parseDates(c)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_DATE_FORMAT")
		return
	}

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		LinkID:    &linkID,
		StartDate: start,
		EndDate:   end,
	}

	report, err := h.analyticsService.GetLinkAnalytics(c.Request.Context(), q)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LINK_ANALYTICS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Link analytics compiled successfully", report)
}

func (h *AnalyticsHandler) parseDates(c *gin.Context) (time.Time, time.Time, error) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("start_date format must be RFC3339")
		}
	} else {
		start = time.Now().Add(-30 * 24 * time.Hour) // 30 days ago fallback default
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("end_date format must be RFC3339")
		}
	} else {
		end = time.Now() // now fallback default
	}

	return start.UTC(), end.UTC(), nil
}
