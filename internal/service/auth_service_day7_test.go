package service

import (
	"context"
	"testing"
	"time"

	"linkpulse/internal/models"

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
