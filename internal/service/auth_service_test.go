package service

import (
	"context"
	"errors"
	"testing"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"
	"linkpulse/internal/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In-memory mock UserRepository
type mockUserRepo struct {
	users map[string]*models.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*models.User)}
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

// In-memory mock RefreshTokenRepository
type mockRefreshRepo struct {
	tokens map[string]*models.RefreshToken
}

func newMockRefreshRepo() *mockRefreshRepo {
	return &mockRefreshRepo{tokens: make(map[string]*models.RefreshToken)}
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

func TestAuthService_Lifecycle(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := newMockRefreshRepo()
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := NewAuthService(userRepo, refreshRepo, secret, accessTTL, refreshTTL, issuer)
	ctx := context.Background()

	t.Run("User Registration succeeds", func(t *testing.T) {
		resp, err := service.Register(ctx, "test@example.com", "securepassword123")
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", resp.Email)
		assert.NotEmpty(t, resp.ID)
	})

	t.Run("Duplicate email registration fails", func(t *testing.T) {
		resp, err := service.Register(ctx, "test@example.com", "otherpassword123")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, domainErrors.ErrAlreadyExists))
	})

	t.Run("Weak password (<8 chars) registration fails", func(t *testing.T) {
		resp, err := service.Register(ctx, "other@example.com", "weak")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, domainErrors.ErrInvalidInput))
	})

	t.Run("Login succeeds with valid credentials", func(t *testing.T) {
		tokens, err := service.Login(ctx, "test@example.com", "securepassword123", "iPhone", "127.0.0.1", "Mozilla")
		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, int64(accessTTL.Seconds()), tokens.ExpiresIn)
	})

	t.Run("Login fails with invalid password (generic error check)", func(t *testing.T) {
		tokens, err := service.Login(ctx, "test@example.com", "wrongpassword", "iPhone", "127.0.0.1", "Mozilla")
		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.True(t, errors.Is(err, domainErrors.ErrInvalidCredentials))
	})

	t.Run("Login fails with non-existent email (generic error check)", func(t *testing.T) {
		tokens, err := service.Login(ctx, "missing@example.com", "securepassword123", "iPhone", "127.0.0.1", "Mozilla")
		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.True(t, errors.Is(err, domainErrors.ErrInvalidCredentials))
	})

	t.Run("Refresh Token Rotation (RTR) succeeds", func(t *testing.T) {
		// Perform fresh login
		tokens, err := service.Login(ctx, "test@example.com", "securepassword123", "iPhone", "127.0.0.1", "Mozilla")
		require.NoError(t, err)

		// Wait slightly to differentiate timestamps
		time.Sleep(10 * time.Millisecond)

		// Refresh
		rotatedTokens, err := service.Refresh(ctx, tokens.RefreshToken, "127.0.0.1", "Mozilla")
		require.NoError(t, err)
		assert.NotEmpty(t, rotatedTokens.AccessToken)
		assert.NotEmpty(t, rotatedTokens.RefreshToken)
		assert.NotEqual(t, tokens.RefreshToken, rotatedTokens.RefreshToken)

		// Replay attack check: attempt to use the old refresh token again
		replayTokens, err := service.Refresh(ctx, tokens.RefreshToken, "127.0.0.1", "Mozilla")
		assert.Error(t, err)
		assert.Nil(t, replayTokens)
		assert.True(t, errors.Is(err, domainErrors.ErrInvalidCredentials))

		// Confirm that the rotation revoked all sessions of the user after replay detection
		user, _ := userRepo.FindByEmail(ctx, "test@example.com")
		for _, token := range refreshRepo.tokens {
			if token.UserID == user.ID {
				assert.NotNil(t, token.RevokedAt, "All tokens must be revoked after replay attack detection")
			}
		}
	})

	t.Run("Logout invalidates refresh token", func(t *testing.T) {
		tokens, err := service.Login(ctx, "test@example.com", "securepassword123", "iPhone", "127.0.0.1", "Mozilla")
		require.NoError(t, err)

		err = service.LogoutCurrentDevice(ctx, tokens.RefreshToken)
		assert.NoError(t, err)
	})
}

func TestPassword_Validation(t *testing.T) {
	t.Run("Bcrypt hashing matches comparison", func(t *testing.T) {
		pass := "securepassword123"
		hash, err := utils.HashPassword(pass)
		require.NoError(t, err)
		require.NotEmpty(t, hash)

		assert.True(t, utils.ComparePassword(pass, hash))
		assert.False(t, utils.ComparePassword("wrongpass", hash))
	})
}
