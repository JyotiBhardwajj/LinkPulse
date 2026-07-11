// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"
	"errors"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository defines database operations for session tracking tokens.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, hash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// Create persists a refresh token record.
func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindByHash retrieves an active or revoked refresh token by its SHA-256 hash.
func (r *refreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).First(&token, "token_hash = ?", hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &token, nil
}

// Revoke invalidates a single refresh token by setting revoked_at to now.
func (r *refreshTokenRepository) Revoke(ctx context.Context, hash string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", &now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainErrors.ErrNotFound
	}
	return nil
}

// RevokeAllForUser invalidates all active sessions for a specific user ID in a single transaction.
func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error
}
