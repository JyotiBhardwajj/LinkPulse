// Package utils provides common helper functions.
package utils

import (
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

// SendError sends a standardized error JSON response.
func SendError(c *gin.Context, status int, message string, code string) {
	c.JSON(status, APIResponse{
		Success: false,
		Error: &ErrorData{
			Code:    code,
			Message: message,
		},
	})
}
