// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"
	"errors"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"

	"gorm.io/gorm"
)

// LinkRepository defines database operations for links.
type LinkRepository interface {
	Create(ctx context.Context, link *models.Link) error
	FindByCode(ctx context.Context, code string) (*models.Link, error)
	FindByOriginalURL(ctx context.Context, originalURL string) (*models.Link, error)
	Deactivate(ctx context.Context, code string) error
}

type linkRepository struct {
	db *gorm.DB
}

// NewLinkRepository creates a new LinkRepository.
func NewLinkRepository(db *gorm.DB) LinkRepository {
	return &linkRepository{db: db}
}

// Create inserts a new link record into the database.
func (r *linkRepository) Create(ctx context.Context, link *models.Link) error {
	if err := r.db.WithContext(ctx).Create(link).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domainErrors.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// FindByCode finds a link by its unique short code.
func (r *linkRepository) FindByCode(ctx context.Context, code string) (*models.Link, error) {
	var link models.Link
	if err := r.db.WithContext(ctx).First(&link, "short_code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &link, nil
}

// FindByOriginalURL finds a link by its original URL.
func (r *linkRepository) FindByOriginalURL(ctx context.Context, originalURL string) (*models.Link, error) {
	var link models.Link
	if err := r.db.WithContext(ctx).First(&link, "original_url = ?", originalURL).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &link, nil
}

// Deactivate soft-deactivates a link by setting is_active to false.
func (r *linkRepository) Deactivate(ctx context.Context, code string) error {
	result := r.db.WithContext(ctx).Model(&models.Link{}).
		Where("short_code = ?", code).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainErrors.ErrNotFound
	}
	return nil
}
