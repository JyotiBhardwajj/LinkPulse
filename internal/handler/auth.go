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

// AuthHandler manages session credentials and rotations.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler instantiates an AuthHandler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register registers a new user account.
//
// @Summary      Register a new user
// @Description  Creates a new user account with the given email and password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.UserRegisterRequest  true  "Registration payload"
// @Success      201      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      409      {object}  models.ErrorResponse
// @Failure      422      {object}  models.ValidationErrorResponse
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
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

// Login authenticates a user and returns a JWT token pair.
//
// @Summary      Login
// @Description  Validates credentials and returns an access token and refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "Login credentials"
// @Success      200      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      422      {object}  models.ValidationErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
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
//
// @Summary      Refresh tokens
// @Description  Accepts a valid refresh token and returns a new access token and refresh token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RefreshRequest  true  "Refresh token payload"
// @Success      200      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      422      {object}  models.ValidationErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
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
//
// @Summary      Logout current session
// @Description  Revokes the provided refresh token, ending the current device session.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      models.RefreshRequest  true  "Refresh token to revoke"
// @Success      200      {object}  models.SuccessResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, utils.FormatValidationErrors(err, req))
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

// GetSessions lists all active sessions for the authenticated user.
//
// @Summary      List active sessions
// @Description  Returns all active device sessions for the currently authenticated user.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.SuccessResponse
// @Failure      401  {object}  models.ErrorResponse
// @Router       /api/v1/auth/sessions [get]
func (h *AuthHandler) GetSessions(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "User context not found in request", "UNAUTHORIZED")
		return
	}

	resp, err := h.authService.GetSessions(c.Request.Context(), authCtx.UserID, authCtx.SessionID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "GET_SESSIONS_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "Active sessions retrieved successfully", resp)
}

// LogoutAll revokes all refresh tokens for the authenticated user.
//
// @Summary      Logout all sessions
// @Description  Revokes all refresh tokens, ending all device sessions for the authenticated user.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.SuccessResponse
// @Failure      401  {object}  models.ErrorResponse
// @Router       /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	authCtx, ok := middleware.GetAuthContext(c)
	if !ok {
		utils.SendError(c, http.StatusUnauthorized, "User context not found in request", "UNAUTHORIZED")
		return
	}

	err := h.authService.LogoutAll(c.Request.Context(), authCtx.UserID)
	if err != nil {
		status := domainErrors.MapToHTTPStatus(err)
		utils.SendError(c, status, err.Error(), "LOGOUT_ALL_FAILED")
		return
	}

	utils.SendSuccess(c, http.StatusOK, "All sessions revoked successfully", nil)
}
