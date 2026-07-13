// Package handler implements HTTP controllers and request parsers.
package handler

import (
	"net/http"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/middleware"
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

// Me retrieves the profile of the currently authenticated user.
//
// @Summary      Get current user profile
// @Description  Returns the profile information of the currently authenticated user.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.SuccessResponse
// @Failure      401  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /api/v1/users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "User context not found in request", "UNAUTHORIZED")
		return
	}

	resp, err := h.userService.GetByID(c.Request.Context(), authCtx.UserID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "USER_NOT_FOUND")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "User profile retrieved successfully", resp)
}
