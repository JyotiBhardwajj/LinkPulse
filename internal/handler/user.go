// Package handler implements HTTP controllers and request parsers.
package handler

import (
	"net/http"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"
	"linkpulse/internal/service"
	"linkpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler manages user profile operations.
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler instantiates a UserHandler.
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register registers a new user credentials.
func (h *UserHandler) Register(c *gin.Context) {
	var req models.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_REQUEST_BODY")
		return
	}

	resp, err := h.userService.Register(c.Request.Context(), req)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "REGISTRATION_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusCreated, "User registered successfully", resp)
}
