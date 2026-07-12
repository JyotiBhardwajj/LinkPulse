// Package utils provides common helper functions.
package utils

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse represents the standardized JSON structure for all API outputs.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

// ErrorData encapsulates structured error details for clients.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendSuccess sends a standardized 2xx/3xx JSON response.
func SendSuccess(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SendError sends a standardized error JSON response, masking internal error details.
func SendError(c *gin.Context, status int, message string, code string) {
	if status == http.StatusInternalServerError {
		reqID := ""
		if val, exists := c.Get("RequestID"); exists {
			if id, ok := val.(string); ok {
				reqID = id
			}
		}

		// Log full internal error with RequestID metadata
		slog.Error("Internal system failure",
			slog.String("request_id", reqID),
			slog.String("error_details", message),
			slog.String("code", code),
		)

		// Sanitize client message to prevent system schema leaks
		message = "An unexpected error occurred. Please try again later."
	}

	c.JSON(status, APIResponse{
		Success: false,
		Error: &ErrorData{
			Code:    code,
			Message: message,
		},
	})
}
