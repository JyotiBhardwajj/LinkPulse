package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/metrics"
	"linkpulse/internal/middleware"
	"linkpulse/internal/models"
	"linkpulse/internal/repository"
	"linkpulse/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Re-use in-memory mock repositories for handler testing
type mockUserRepo struct {
	users map[string]*models.User
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	if _, exists := m.users[user.Email]; exists {
		return domainErrors.ErrAlreadyExists
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domainErrors.ErrNotFound
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	u, exists := m.users[email]
	if !exists {
		return nil, domainErrors.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *models.User) error {
	m.users[user.Email] = user
	return nil
}

type mockRefreshRepo struct {
	tokens map[string]*models.RefreshToken
}

func (m *mockRefreshRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockRefreshRepo) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	t, exists := m.tokens[hash]
	if !exists {
		return nil, domainErrors.ErrNotFound
	}
	return t, nil
}

func (m *mockRefreshRepo) Revoke(ctx context.Context, hash string) error {
	t, exists := m.tokens[hash]
	if !exists {
		return domainErrors.ErrNotFound
	}
	now := time.Now()
	t.RevokedAt = &now
	return nil
}

func (m *mockRefreshRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]models.RefreshToken, error) {
	var active []models.RefreshToken
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == userID && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			active = append(active, *t)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		tI := active[i].LastUsedAt
		if tI.IsZero() {
			tI = active[i].CreatedAt
		}
		tJ := active[j].LastUsedAt
		if tJ.IsZero() {
			tJ = active[j].CreatedAt
		}
		if tI.Equal(tJ) {
			return active[i].CreatedAt.Before(active[j].CreatedAt)
		}
		return tI.Before(tJ)
	})
	return active, nil
}

type mockUserService struct {
	userRepo *mockUserRepo
}

func (s *mockUserService) Register(ctx context.Context, req models.UserRegisterRequest) (*models.UserResponse, error) {
	user := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	_ = s.userRepo.Create(ctx, user)
	return &models.UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}, nil
}

func (s *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*models.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.UserProfileResponse{ID: user.ID, Email: user.Email, Role: user.Role, CreatedAt: user.CreatedAt}, nil
}

type mockTxManager struct {
	userRepo    *mockUserRepo
	refreshRepo *mockRefreshRepo
}

func (m *mockTxManager) WithinTransaction(ctx context.Context, fn func(txRepo repository.RepositoryManager) error) error {
	repoMgr := &mockRepoManager{
		userRepo:    m.userRepo,
		refreshRepo: m.refreshRepo,
	}
	return fn(repoMgr)
}

type mockRepoManager struct {
	userRepo    *mockUserRepo
	refreshRepo *mockRefreshRepo
}

func (m *mockRepoManager) Users() repository.UserRepository {
	return m.userRepo
}

func (m *mockRepoManager) Links() repository.LinkRepository {
	return nil
}

func (m *mockRepoManager) Analytics() repository.AnalyticsRepository {
	return nil
}

func (m *mockRepoManager) RefreshTokens() repository.RefreshTokenRepository {
	return m.refreshRepo
}

func TestAuthHandler_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockUserRepo{users: make(map[string]*models.User)}
	refreshRepo := &mockRefreshRepo{tokens: make(map[string]*models.RefreshToken)}
	txMgr := &mockTxManager{userRepo: userRepo, refreshRepo: refreshRepo}
	secret := "handlertestsecretkeythatisreallylong"
	issuer := "linkpulse-api"
	accessTTL := 5 * time.Minute
	refreshTTL := 24 * time.Hour

	authService := service.NewAuthService(userRepo, refreshRepo, txMgr, secret, accessTTL, refreshTTL, issuer, 10, metrics.NewNoOpMetrics())
	mockUS := &mockUserService{userRepo: userRepo}

	// Handlers
	authHandler := NewAuthHandler(authService)
	userHandler := NewUserHandler(mockUS)

	// Router setup
	r := gin.New()
	authMiddleware := middleware.Auth(secret, issuer)

	api := r.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/logout", authMiddleware, authHandler.Logout)
		}
		users := api.Group("/users")
		{
			users.GET("/me", authMiddleware, userHandler.Me)
		}
	}

	t.Run("POST /register succeeds", func(t *testing.T) {
		reqBody, _ := json.Marshal(models.UserRegisterRequest{
			Email:    "handler@example.com",
			Password: "securepassword123",
		})
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "User registered successfully")
	})

	t.Run("POST /login succeeds, Me retrieves profile, and Logout revokes token", func(t *testing.T) {
		// Register user first
		_, _ = authService.Register(context.Background(), "login@example.com", "securepassword123")

		reqBody, _ := json.Marshal(models.LoginRequest{
			Email:    "login@example.com",
			Password: "securepassword123",
		})
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp.Data.AccessToken)

		// Test GET /me with valid token
		reqMe, _ := http.NewRequest("GET", "/api/v1/users/me", nil)
		reqMe.Header.Set("Authorization", "Bearer "+resp.Data.AccessToken)
		wMe := httptest.NewRecorder()

		r.ServeHTTP(wMe, reqMe)
		assert.Equal(t, http.StatusOK, wMe.Code)
		assert.Contains(t, wMe.Body.String(), "login@example.com")

		// Test POST /logout (authenticated endpoint - Option A)
		logoutBody, _ := json.Marshal(models.RefreshRequest{
			RefreshToken: resp.Data.RefreshToken,
		})
		reqLogout, _ := http.NewRequest("POST", "/api/v1/auth/logout", bytes.NewBuffer(logoutBody))
		reqLogout.Header.Set("Authorization", "Bearer "+resp.Data.AccessToken)
		reqLogout.Header.Set("Content-Type", "application/json")
		wLogout := httptest.NewRecorder()

		r.ServeHTTP(wLogout, reqLogout)
		assert.Equal(t, http.StatusOK, wLogout.Code)
		assert.Contains(t, wLogout.Body.String(), "Logout successful")

		// Test GET /me with invalid Bearer prefix
		reqMeBad, _ := http.NewRequest("GET", "/api/v1/users/me", nil)
		reqMeBad.Header.Set("Authorization", "BadPrefix "+resp.Data.AccessToken)
		wMeBad := httptest.NewRecorder()

		r.ServeHTTP(wMeBad, reqMeBad)
		assert.Equal(t, http.StatusUnauthorized, wMeBad.Code)
	})
}
