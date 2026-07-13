package handler

import (
	"net/http"
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

	var req models.AnalyticsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	start, end, _ := h.parseDates(req)
	if req.StartDate != "" && req.EndDate != "" && end.Before(start) {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "end_date",
				Rule:    "gtfield",
				Message: "The end_date field must be after or equal to start_date",
			},
		})
		return
	}

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		StartDate: start,
		EndDate:   end,
		Interval:  req.Interval,
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

	var req models.AnalyticsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	q := models.AnalyticsQuery{
		UserID: authCtx.UserID,
		Limit:  req.Limit,
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

	var req models.AnalyticsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	start, end, _ := h.parseDates(req)
	if req.StartDate != "" && req.EndDate != "" && end.Before(start) {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "end_date",
				Rule:    "gtfield",
				Message: "The end_date field must be after or equal to start_date",
			},
		})
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

	var req models.AnalyticsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	start, end, _ := h.parseDates(req)
	if req.StartDate != "" && req.EndDate != "" && end.Before(start) {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "end_date",
				Rule:    "gtfield",
				Message: "The end_date field must be after or equal to start_date",
			},
		})
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

	var req models.AnalyticsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	start, end, _ := h.parseDates(req)
	if req.StartDate != "" && req.EndDate != "" && end.Before(start) {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "end_date",
				Rule:    "gtfield",
				Message: "The end_date field must be after or equal to start_date",
			},
		})
		return
	}

	q := models.AnalyticsQuery{
		UserID:    authCtx.UserID,
		StartDate: start,
		EndDate:   end,
		Limit:     req.Limit,
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

	var req models.AnalyticsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
		return
	}

	start, end, _ := h.parseDates(req)
	if req.StartDate != "" && req.EndDate != "" && end.Before(start) {
		utils.SendValidationError(c, []models.ValidationError{
			{
				Field:   "end_date",
				Rule:    "gtfield",
				Message: "The end_date field must be after or equal to start_date",
			},
		})
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

func (h *AnalyticsHandler) parseDates(req models.AnalyticsQueryRequest) (time.Time, time.Time, error) {
	var start, end time.Time
	var err error

	if req.StartDate != "" {
		start, err = time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	} else {
		start = time.Now().Add(-30 * 24 * time.Hour)
	}

	if req.EndDate != "" {
		end, err = time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	} else {
		end = time.Now()
	}

	return start.UTC(), end.UTC(), nil
}
