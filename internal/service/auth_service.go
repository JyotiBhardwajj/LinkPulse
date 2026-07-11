// Package service implements core domain business logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"linkpulse/internal/auth"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
)

// AuthService defines operations for user registration, login, and session rotation.
type AuthService interface {
	Register(ctx context.Context, email string, password string) (*models.UserResponse, error)
	Login(ctx context.Context, email string, password string, deviceName string, ip string, userAgent string) (*models.TokenResponse, error)
	Refresh(ctx context.Context, refreshToken string, ip string, userAgent string) (*models.TokenResponse, error)
	LogoutCurrentDevice(ctx context.Context, refreshToken string) error
	LogoutAllDevices(ctx context.Context, userID uuid.UUID) error
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error)
}

type authService struct {
	userRepo        repository.UserRepository
	refreshRepo     repository.RefreshTokenRepository
	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	tokenIssuer     string
}

// NewAuthService instantiates a new AuthService implementation.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	tokenIssuer string,
) AuthService {
	return &authService{
		userRepo:        userRepo,
		refreshRepo:     refreshRepo,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		tokenIssuer:     tokenIssuer,
	}
}

// Register registers a new user credentials, enforcing password length limits.
func (s *authService) Register(ctx context.Context, email string, password string) (*models.UserResponse, error) {
	if len(password) < 8 || len(password) > 72 {
		return nil, fmt.Errorf("%w: password must be between 8 and 72 characters", domainErrors.ErrInvalidInput)
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// Login validates user password and returns access/refresh tokens.
func (s *authService) Login(ctx context.Context, email string, password string, deviceName string, ip string, userAgent string) (*models.TokenResponse, error) {
	// Generic error prevents user enumeration
	genericErr := domainErrors.ErrInvalidCredentials

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNotFound) {
			return nil, genericErr
		}
		return nil, err
	}

	if !utils.ComparePassword(password, user.PasswordHash) {
		return nil, genericErr
	}

	// Sign Access Token
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Email, s.jwtSecret, s.accessTokenTTL, s.tokenIssuer)
	if err != nil {
		return nil, err
	}

	// Generate Refresh Token
	rawRefreshToken, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	hashedToken := utils.HashSHA256(rawRefreshToken)
	ipHash := utils.HashIP(ip)

	sessionToken := &models.RefreshToken{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  hashedToken,
		DeviceName: deviceName,
		IPHash:     ipHash,
		UserAgent:  userAgent,
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(s.refreshTokenTTL),
		CreatedAt:  time.Now(),
	}

	if err := s.refreshRepo.Create(ctx, sessionToken); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// Refresh rotates the refresh token and returns a new access/refresh pair.
func (s *authService) Refresh(ctx context.Context, refreshToken string, ip string, userAgent string) (*models.TokenResponse, error) {
	hashedToken := utils.HashSHA256(refreshToken)

	token, err := s.refreshRepo.FindByHash(ctx, hashedToken)
	if err != nil {
		return nil, domainErrors.ErrInvalidCredentials
	}

	// RTR Replay attack detection: If the refresh token was already revoked, invalidate the user's entire session list.
	if token.RevokedAt != nil {
		slog.Warn("Replay attack detected on already revoked refresh token. Revoking all sessions for user.", "user_id", token.UserID)
		_ = s.refreshRepo.RevokeAllForUser(ctx, token.UserID)
		return nil, domainErrors.ErrInvalidCredentials
	}

	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return nil, domainErrors.ErrInvalidCredentials
	}

	// Invalidate the current refresh token
	if err := s.refreshRepo.Revoke(ctx, hashedToken); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, domainErrors.ErrInvalidCredentials
	}

	// Generate Access Token B
	newAccessToken, err := auth.GenerateAccessToken(user.ID, user.Email, s.jwtSecret, s.accessTokenTTL, s.tokenIssuer)
	if err != nil {
		return nil, err
	}

	// Generate Refresh Token B
	rawRefreshTokenB, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	hashedTokenB := utils.HashSHA256(rawRefreshTokenB)
	ipHash := utils.HashIP(ip)

	sessionTokenB := &models.RefreshToken{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  hashedTokenB,
		DeviceName: token.DeviceName,
		IPHash:     ipHash,
		UserAgent:  userAgent,
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(s.refreshTokenTTL),
		CreatedAt:  time.Now(),
	}

	if err := s.refreshRepo.Create(ctx, sessionTokenB); err != nil {
		return nil, fmt.Errorf("failed to save rotated refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: rawRefreshTokenB,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// LogoutCurrentDevice invalidates the active refresh token.
func (s *authService) LogoutCurrentDevice(ctx context.Context, refreshToken string) error {
	hashedToken := utils.HashSHA256(refreshToken)
	err := s.refreshRepo.Revoke(ctx, hashedToken)
	if err != nil && errors.Is(err, domainErrors.ErrNotFound) {
		return domainErrors.ErrInvalidCredentials
	}
	return err
}

// LogoutAllDevices invalidates all sessions for a user.
func (s *authService) LogoutAllDevices(ctx context.Context, userID uuid.UUID) error {
	return s.refreshRepo.RevokeAllForUser(ctx, userID)
}

// GetCurrentUser returns user details by UUID.
func (s *authService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}
