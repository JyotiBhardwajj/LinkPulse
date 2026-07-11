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

// AuthHandler manages session credentials and rotations.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler instantiates an AuthHandler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register registers a new user credentials.
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_REQUEST_BODY")
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "REGISTRATION_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusCreated, "User registered successfully", resp)
}

// Login checks credentials and generates token pairs.
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_REQUEST_BODY")
		return
	}

	// Extract device name and network metadata
	deviceName := c.GetHeader("X-Device-Name")
	if deviceName == "" {
		deviceName = "Unknown Device"
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, deviceName, ip, userAgent)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "INVALID_CREDENTIALS")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Login successful", resp)
}

// Refresh rotates the refresh token and returns a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_REQUEST_BODY")
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	resp, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken, ip, userAgent)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "INVALID_REFRESH_TOKEN")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Token refreshed successfully", resp)
}

// Logout revokes the refresh token of the current session.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, err.Error(), "INVALID_REQUEST_BODY")
		return
	}

	err := h.authService.LogoutCurrentDevice(c.Request.Context(), req.RefreshToken)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LOGOUT_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Logout successful", nil)
}
