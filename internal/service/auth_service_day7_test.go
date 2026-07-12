package service

import (
	"context"
	"testing"
	"time"

	"linkpulse/internal/models"
	"linkpulse/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_SessionPruningAndPruningOrder(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := newMockRefreshRepo()
	txMgr := &mockTxManager{userRepo: userRepo, refreshRepo: refreshRepo}
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	// Limit to max 3 sessions
	service := NewAuthService(userRepo, refreshRepo, txMgr, secret, accessTTL, refreshTTL, issuer, 3)
	ctx := context.Background()

	// Register user
	registerResp, err := service.Register(ctx, "sessionpruning@example.com", "securepassword123")
	require.NoError(t, err)

	// User logs in 4 times
	// Session 1
	login1, err := service.Login(ctx, "sessionpruning@example.com", "securepassword123", "iPhone", "192.168.1.1", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond) // Ensure unique timestamps

	// Session 2
	login2, err := service.Login(ctx, "sessionpruning@example.com", "securepassword123", "Windows Desktop", "192.168.1.2", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.0.0 Safari/537.36")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// Session 3
	login3, err := service.Login(ctx, "sessionpruning@example.com", "securepassword123", "MacBook", "192.168.1.3", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Safari/605.1.15")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// Verify we have 3 active sessions
	active1, err := refreshRepo.FindActiveByUserID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Len(t, active1, 3)

	// Session 4 (triggers pruning of oldest session - Session 1)
	_, err = service.Login(ctx, "sessionpruning@example.com", "securepassword123", "Android Tablet", "192.168.1.4", "Mozilla/5.0 (Linux; Android 11; Tab) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.0.0 Safari/537.36")
	require.NoError(t, err)

	// Verify total active sessions remains 3
	active2, err := refreshRepo.FindActiveByUserID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Len(t, active2, 3)

	// Retrieve session responses to check parsed user-agents
	sessions, err := service.GetSessions(ctx, registerResp.ID, uuid.Nil)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Verify Session 1 (iPhone login) is pruned, and we have Windows, Mac, Android sessions
	devices := make(map[string]bool)
	for _, s := range sessions {
		devices[s.Device] = true
	}
	assert.False(t, devices["iPhone (Mobile)"])
	assert.True(t, devices["Windows Desktop (Desktop)"])
	assert.True(t, devices["MacBook (Desktop)"])
	assert.True(t, devices["Android Tablet (Mobile)"] || devices["Android Tablet (Desktop)"])

	_ = login1
	_ = login2
	_ = login3
}

func TestAuthService_RolePromotionAndDemotion(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := newMockRefreshRepo()
	txMgr := &mockTxManager{userRepo: userRepo, refreshRepo: refreshRepo}
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := NewAuthService(userRepo, refreshRepo, txMgr, secret, accessTTL, refreshTTL, issuer, 10)
	ctx := context.Background()

	// Register user
	registerResp, err := service.Register(ctx, "rolechange@example.com", "securepassword123")
	require.NoError(t, err)

	// Get user from database, role should be User
	user, err := userRepo.FindByID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleUser, user.Role)

	// Promote user to admin
	err = service.PromoteUser(ctx, registerResp.ID)
	require.NoError(t, err)

	// Verify role is Admin
	user, err = userRepo.FindByID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleAdmin, user.Role)

	// Demote user to user
	err = service.DemoteUser(ctx, registerResp.ID)
	require.NoError(t, err)

	// Verify role is User again
	user, err = userRepo.FindByID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleUser, user.Role)
}

func TestAuthService_LogoutAll(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := newMockRefreshRepo()
	txMgr := &mockTxManager{userRepo: userRepo, refreshRepo: refreshRepo}
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := NewAuthService(userRepo, refreshRepo, txMgr, secret, accessTTL, refreshTTL, issuer, 10)
	ctx := context.Background()

	// Register user
	registerResp, err := service.Register(ctx, "logoutall@example.com", "securepassword123")
	require.NoError(t, err)

	// Log in twice
	_, err = service.Login(ctx, "logoutall@example.com", "securepassword123", "Device1", "127.0.0.1", "Mozilla/5.0")
	require.NoError(t, err)
	_, err = service.Login(ctx, "logoutall@example.com", "securepassword123", "Device2", "127.0.0.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Verify we have 2 active sessions
	active, err := refreshRepo.FindActiveByUserID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Len(t, active, 2)

	// Logout all
	err = service.LogoutAll(ctx, registerResp.ID)
	require.NoError(t, err)

	// Verify we have 0 active sessions
	activeAfter, err := refreshRepo.FindActiveByUserID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Len(t, activeAfter, 0)
}

func TestAuthService_CurrentSessionDetection(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := newMockRefreshRepo()
	txMgr := &mockTxManager{userRepo: userRepo, refreshRepo: refreshRepo}
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := NewAuthService(userRepo, refreshRepo, txMgr, secret, accessTTL, refreshTTL, issuer, 10)
	ctx := context.Background()

	// Register and login
	registerResp, err := service.Register(ctx, "currentsess@example.com", "securepassword123")
	require.NoError(t, err)

	_, err = service.Login(ctx, "currentsess@example.com", "securepassword123", "Device1", "127.0.0.1", "Mozilla/5.0")
	require.NoError(t, err)

	// Find the session ID we generated
	tokens, err := refreshRepo.FindActiveByUserID(ctx, registerResp.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	sessionID := tokens[0].ID

	// Verify GetSessions detects current session correctly when currentTokenID is passed
	sessions, err := service.GetSessions(ctx, registerResp.ID, sessionID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.True(t, sessions[0].CurrentSession)

	// Verify GetSessions does not detect current session if wrong session ID passed
	sessionsOther, err := service.GetSessions(ctx, registerResp.ID, uuid.New())
	require.NoError(t, err)
	require.Len(t, sessionsOther, 1)
	assert.False(t, sessionsOther[0].CurrentSession)
}

type rollbackMockTxManager struct {
	userRepo    *mockUserRepo
	refreshRepo *mockRefreshRepo
}

func (m *rollbackMockTxManager) WithinTransaction(ctx context.Context, fn func(txRepo repository.RepositoryManager) error) error {
	// Clone users map
	userBackup := make(map[string]*models.User)
	for k, v := range m.userRepo.users {
		valCopy := *v
		userBackup[k] = &valCopy
	}

	// Clone tokens map
	tokenBackup := make(map[string]*models.RefreshToken)
	for k, v := range m.refreshRepo.tokens {
		valCopy := *v
		tokenBackup[k] = &valCopy
	}

	err := fn(&mockRepoManager{
		userRepo:    m.userRepo,
		refreshRepo: m.refreshRepo,
	})

	if err != nil {
		// Rollback: restore state
		m.userRepo.users = userBackup
		m.refreshRepo.tokens = tokenBackup
	}

	return err
}

func TestAuthService_TransactionRollback(t *testing.T) {
	userRepo := newMockUserRepo()
	refreshRepo := newMockRefreshRepo()
	txMgr := &rollbackMockTxManager{userRepo: userRepo, refreshRepo: refreshRepo}
	secret := "supersecretjwtkeythatisreallylongandsecure"
	issuer := "linkpulse-api"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := NewAuthService(userRepo, refreshRepo, txMgr, secret, accessTTL, refreshTTL, issuer, 10)
	ctx := context.Background()

	// Register user
	registerResp, err := service.Register(ctx, "rollback@example.com", "securepassword123")
	require.NoError(t, err)

	// Verify initial role is User
	user, err := userRepo.FindByID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleUser, user.Role)

	// Modify promote user to fail inside transaction or trigger error
	err = txMgr.WithinTransaction(ctx, func(txRepo repository.RepositoryManager) error {
		user.Role = models.RoleAdmin
		_ = txRepo.Users().Update(ctx, user)
		// Return intentional error to trigger rollback
		return assert.AnError
	})
	assert.Error(t, err)

	// Verify role was NOT updated (remained RoleUser) due to rollback!
	userAfter, err := userRepo.FindByID(ctx, registerResp.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleUser, userAfter.Role)
}
