// Package utils provides common helper functions.
package utils

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"linkpulse/internal/models"

	"github.com/gin-gonic/gin"
)

// getRequestID retrieves Request ID from Gin context.
func getRequestID(c *gin.Context) string {
	if val, exists := c.Get("RequestID"); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return c.GetHeader("X-Request-ID")
}

// SendSuccess sends a standardized 2xx/3xx JSON response.
func SendSuccess(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, models.SuccessResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: getRequestID(c),
	})
}

// SendPaginatedSuccess sends a standardized list JSON response.
func SendPaginatedSuccess(c *gin.Context, status int, message string, data interface{}, page, pageSize int, total int64, totalPages int) {
	c.JSON(status, models.PaginatedResponse{
		Success: true,
		Message: message,
		Data:    data,
		Metadata: models.PaginationMetadata{
			Page:        page,
			PageSize:    pageSize,
			Total:       total,
			TotalPages:  totalPages,
			HasNext:     page < totalPages,
			HasPrevious: page > 1,
		},
		RequestID: getRequestID(c),
	})
}

// SendError sends a standardized error JSON response, using the RFC7807 problem details specification.
func SendError(c *gin.Context, status int, message string, code string) {
	reqID := getRequestID(c)

	if status == http.StatusInternalServerError {
		// Log full internal error with RequestID metadata
		slog.Error("Internal system failure",
			slog.String("request_id", reqID),
			slog.String("error_details", message),
			slog.String("code", code),
		)

		// Sanitize client message to prevent system schema leaks
		message = "An unexpected error occurred. Please try again later."
	}

	// Dynamic RFC7807 properties mapping
	errType := fmt.Sprintf("https://linkpulse.com/errors/%s", strings.ToLower(strings.ReplaceAll(code, "_", "-")))
	errTitle := strings.Title(strings.ToLower(strings.ReplaceAll(code, "_", " ")))

	// Explicitly set RFC7807 content type header
	c.Header("Content-Type", "application/problem+json")

	c.JSON(status, models.ErrorResponse{
		Success: false,
		Error: &models.ErrorDetails{
			Code:    code,
			Message: message,
		},
		Type:      errType,
		Title:     errTitle,
		Status:    status,
		Detail:    message,
		Instance:  c.Request.URL.Path,
		RequestID: reqID,
	})
}

// SendValidationError sends a standardized validation failure response (HTTP 422).
func SendValidationError(c *gin.Context, details []models.ValidationError) {
	reqID := getRequestID(c)

	// Explicitly set RFC7807 content type header
	c.Header("Content-Type", "application/problem+json")

	c.JSON(http.StatusUnprocessableEntity, models.ValidationErrorResponse{
		Success: false,
		Error: &models.ErrorDetails{
			Code:    "VALIDATION_ERROR",
			Message: "Validation failed",
		},
		Type:      "https://linkpulse.com/errors/validation-error",
		Title:     "Validation Error",
		Status:    http.StatusUnprocessableEntity,
		Detail:    "One or more fields failed validation constraints.",
		Instance:  c.Request.URL.Path,
		Details:   details,
		RequestID: reqID,
	})
}
