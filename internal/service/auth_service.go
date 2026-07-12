// Package service implements core domain business logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"linkpulse/internal/auth"
	"linkpulse/internal/constants"
	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/logger"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
	"github.com/mssola/user_agent"
)

// AuthService defines operations for user registration, login, and session rotation.
type AuthService interface {
	Register(ctx context.Context, email string, password string) (*models.UserResponse, error)
	Login(ctx context.Context, email string, password string, deviceName string, ip string, userAgent string) (*models.TokenResponse, error)
	Refresh(ctx context.Context, refreshToken string, ip string, userAgent string) (*models.TokenResponse, error)
	LogoutCurrentDevice(ctx context.Context, refreshToken string) error
	LogoutAllDevices(ctx context.Context, userID uuid.UUID) error
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*models.UserProfileResponse, error)

	// Day 7 additions
	GetSessions(ctx context.Context, userID uuid.UUID, currentTokenID uuid.UUID) ([]models.SessionResponse, error)
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	PromoteUser(ctx context.Context, targetUserID uuid.UUID) error
	DemoteUser(ctx context.Context, targetUserID uuid.UUID) error
}

type authService struct {
	userRepo           repository.UserRepository
	refreshRepo        repository.RefreshTokenRepository
	txMgr              repository.TransactionManager
	jwtSecret          string
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
	tokenIssuer        string
	maxSessionsPerUser int
}

// NewAuthService instantiates a new AuthService implementation.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	txMgr repository.TransactionManager,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	tokenIssuer string,
	maxSessionsPerUser int,
) AuthService {
	if maxSessionsPerUser <= 0 {
		maxSessionsPerUser = 10
	}
	return &authService{
		userRepo:           userRepo,
		refreshRepo:        refreshRepo,
		txMgr:              txMgr,
		jwtSecret:          jwtSecret,
		accessTokenTTL:     accessTokenTTL,
		refreshTokenTTL:    refreshTokenTTL,
		tokenIssuer:        tokenIssuer,
		maxSessionsPerUser: maxSessionsPerUser,
	}
}

func getAuditMetadata(ctx context.Context) (reqID string, ipHash string) {
	if val := ctx.Value(constants.RequestIDKey); val != nil {
		if id, ok := val.(string); ok {
			reqID = id
		}
	}
	var ip string
	if val := ctx.Value(constants.ClientIPKey); val != nil {
		if clIP, ok := val.(string); ok {
			ip = clIP
		}
	}
	if ip != "" {
		ipHash = utils.HashIP(ip)
	}
	return
}

// Register registers a new user credentials.
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
		Role:         models.RoleUser, // default role
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

// Login validates user password, prunes active sessions to respect limits, and returns tokens.
func (s *authService) Login(ctx context.Context, email string, password string, deviceName string, ip string, userAgent string) (*models.TokenResponse, error) {
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

	var accessToken string
	var rawRefreshToken string

	// Enforce MAX_SESSIONS_PER_USER inside a database transaction bounds
	err = s.txMgr.WithinTransaction(ctx, func(txRepo repository.RepositoryManager) error {
		activeSessions, err := txRepo.RefreshTokens().FindActiveByUserID(ctx, user.ID)
		if err != nil {
			return err
		}

		// Prune excess sessions ordered by oldest first (FindActiveByUserID returns them sorted ASC)
		if len(activeSessions) >= s.maxSessionsPerUser {
			excess := len(activeSessions) - s.maxSessionsPerUser + 1
			for i := 0; i < excess && i < len(activeSessions); i++ {
				if err := txRepo.RefreshTokens().Revoke(ctx, activeSessions[i].TokenHash); err != nil {
					return err
				}
			}
		}

		accessToken, err = auth.GenerateAccessToken(user.ID, user.Email, user.Role, s.jwtSecret, s.accessTokenTTL, s.tokenIssuer)
		if err != nil {
			return err
		}

		rawRefreshToken, err = auth.GenerateSecureToken()
		if err != nil {
			return err
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
			CreatedIP:  ip,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		return txRepo.RefreshTokens().Create(ctx, sessionToken)
	})

	if err != nil {
		return nil, err
	}

	reqID, ipHash := getAuditMetadata(ctx)
	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     user.ID,
		Event:      logger.EventLogin,
		Resource:   "users",
		ResourceID: user.ID.String(),
		IPHash:     ipHash,
	})

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// Refresh rotates the refresh token and updates device/IP metadata.
func (s *authService) Refresh(ctx context.Context, refreshToken string, ip string, userAgent string) (*models.TokenResponse, error) {
	hashedToken := utils.HashSHA256(refreshToken)

	var newAccessToken string
	var rawRefreshTokenB string
	var userID uuid.UUID

	err := s.txMgr.WithinTransaction(ctx, func(txRepo repository.RepositoryManager) error {
		token, err := txRepo.RefreshTokens().FindByHash(ctx, hashedToken)
		if err != nil {
			return domainErrors.ErrInvalidCredentials
		}

		userID = token.UserID

		// RTR Replay attack check
		if token.RevokedAt != nil {
			slog.Warn("Replay attack detected on already revoked refresh token. Revoking all sessions for user.", "user_id", token.UserID)
			_ = txRepo.RefreshTokens().RevokeAllForUser(ctx, token.UserID)
			return domainErrors.ErrInvalidCredentials
		}

		if time.Now().After(token.ExpiresAt) {
			return domainErrors.ErrInvalidCredentials
		}

		// Revoke old token
		if err := txRepo.RefreshTokens().Revoke(ctx, hashedToken); err != nil {
			return err
		}

		user, err := txRepo.Users().FindByID(ctx, token.UserID)
		if err != nil {
			return domainErrors.ErrInvalidCredentials
		}

		newAccessToken, err = auth.GenerateAccessToken(user.ID, user.Email, user.Role, s.jwtSecret, s.accessTokenTTL, s.tokenIssuer)
		if err != nil {
			return err
		}

		rawRefreshTokenB, err = auth.GenerateSecureToken()
		if err != nil {
			return err
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
			CreatedIP:  token.CreatedIP,
			CreatedAt:  token.CreatedAt,
			UpdatedAt:  time.Now(),
		}

		return txRepo.RefreshTokens().Create(ctx, sessionTokenB)
	})

	if err != nil {
		return nil, err
	}

	reqID, ipHash := getAuditMetadata(ctx)
	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     userID,
		Event:      logger.EventRefresh,
		Resource:   "users",
		ResourceID: userID.String(),
		IPHash:     ipHash,
	})

	return &models.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: rawRefreshTokenB,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// LogoutCurrentDevice invalidates the active refresh token.
func (s *authService) LogoutCurrentDevice(ctx context.Context, refreshToken string) error {
	hashedToken := utils.HashSHA256(refreshToken)
	token, err := s.refreshRepo.FindByHash(ctx, hashedToken)
	if err != nil {
		return domainErrors.ErrInvalidCredentials
	}

	err = s.refreshRepo.Revoke(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNotFound) {
			return domainErrors.ErrInvalidCredentials
		}
		return err
	}

	reqID, ipHash := getAuditMetadata(ctx)
	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     token.UserID,
		Event:      logger.EventLogout,
		Resource:   "users",
		ResourceID: token.UserID.String(),
		IPHash:     ipHash,
	})

	return nil
}

// LogoutAllDevices invalidates all sessions for a user.
func (s *authService) LogoutAllDevices(ctx context.Context, userID uuid.UUID) error {
	return s.LogoutAll(ctx, userID)
}

// GetCurrentUser returns user profile details by UUID.
func (s *authService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*models.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetSessions returns all active sessions for a user, using github.com/mssola/user_agent parsing.
func (s *authService) GetSessions(ctx context.Context, userID uuid.UUID, currentTokenID uuid.UUID) ([]models.SessionResponse, error) {
	tokens, err := s.refreshRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var sessions []models.SessionResponse
	for _, t := range tokens {
		ua := user_agent.New(t.UserAgent)
		browser, browserVer := ua.Browser()
		browserName := fmt.Sprintf("%s %s", browser, browserVer)
		if browser == "" {
			browserName = "Unknown"
		}

		os := ua.OS()
		if os == "" {
			os = "Unknown"
		}

		device := "Desktop"
		if ua.Mobile() {
			device = "Mobile"
		} else if ua.Bot() {
			device = "Bot"
		}

		// Retain friendly custom DeviceName if specified during login
		if t.DeviceName != "" {
			device = fmt.Sprintf("%s (%s)", t.DeviceName, device)
		}

		sessions = append(sessions, models.SessionResponse{
			SessionID:      t.ID,
			Device:         device,
			Browser:        browserName,
			OS:             os,
			IPHash:         t.IPHash,
			LastUsed:       t.LastUsedAt,
			CreatedAt:      t.CreatedAt,
			CurrentSession: t.ID == currentTokenID,
		})
	}

	return sessions, nil
}

// LogoutAll invalidates all sessions for a user.
func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	err := s.refreshRepo.RevokeAllForUser(ctx, userID)
	if err != nil {
		return err
	}

	reqID, ipHash := getAuditMetadata(ctx)
	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     userID,
		Event:      logger.EventLogoutAll,
		Resource:   "users",
		ResourceID: userID.String(),
		IPHash:     ipHash,
	})

	return nil
}

// PromoteUser updates a user's role to Admin inside a transaction.
func (s *authService) PromoteUser(ctx context.Context, targetUserID uuid.UUID) error {
	var userEmail string
	err := s.txMgr.WithinTransaction(ctx, func(txRepo repository.RepositoryManager) error {
		user, err := txRepo.Users().FindByID(ctx, targetUserID)
		if err != nil {
			return err
		}

		userEmail = user.Email
		user.Role = models.RoleAdmin
		user.UpdatedAt = time.Now()

		return txRepo.Users().Update(ctx, user)
	})

	if err != nil {
		return err
	}

	reqID, ipHash := getAuditMetadata(ctx)
	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     targetUserID,
		Event:      logger.EventRoleChange,
		Resource:   "users",
		ResourceID: userEmail,
		IPHash:     ipHash,
	})

	return nil
}

// DemoteUser updates a user's role to User inside a transaction.
func (s *authService) DemoteUser(ctx context.Context, targetUserID uuid.UUID) error {
	var userEmail string
	err := s.txMgr.WithinTransaction(ctx, func(txRepo repository.RepositoryManager) error {
		user, err := txRepo.Users().FindByID(ctx, targetUserID)
		if err != nil {
			return err
		}

		userEmail = user.Email
		user.Role = models.RoleUser
		user.UpdatedAt = time.Now()

		return txRepo.Users().Update(ctx, user)
	})

	if err != nil {
		return err
	}

	reqID, ipHash := getAuditMetadata(ctx)
	logger.GetAuditLogger().Submit(logger.AuditRecord{
		RequestID:  reqID,
		UserID:     targetUserID,
		Event:      logger.EventRoleChange,
		Resource:   "users",
		ResourceID: userEmail,
		IPHash:     ipHash,
	})

	return nil
}
