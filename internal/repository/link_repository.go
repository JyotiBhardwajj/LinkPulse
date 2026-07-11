// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LinkRepository defines database operations for links.
type LinkRepository interface {
	Create(ctx context.Context, link *models.Link) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Link, error)
	FindByShortCode(ctx context.Context, code string) (*models.Link, error)
	FindByUser(ctx context.Context, userID uuid.UUID, query models.ListLinksQuery) ([]models.Link, int64, error)
	Update(ctx context.Context, link *models.Link) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ExistsAlias(ctx context.Context, alias string) (bool, error)
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

// FindByID retrieves a link by its primary key ID.
func (r *linkRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Link, error) {
	var link models.Link
	if err := r.db.WithContext(ctx).First(&link, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &link, nil
}

// FindByShortCode finds a link by its unique short code.
func (r *linkRepository) FindByShortCode(ctx context.Context, code string) (*models.Link, error) {
	var link models.Link
	if err := r.db.WithContext(ctx).First(&link, "short_code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &link, nil
}

// FindByUser retrieves a paginated and filtered list of links owned by the user.
func (r *linkRepository) FindByUser(ctx context.Context, userID uuid.UUID, q models.ListLinksQuery) ([]models.Link, int64, error) {
	var links []models.Link
	var total int64

	db := r.db.WithContext(ctx)

	// If filtering specifically for soft-deleted records, use GORM's Unscoped scope.
	if q.Status == "deleted" {
		db = db.Unscoped().Where("deleted_at IS NOT NULL")
	} else {
		db = db.Model(&models.Link{})
	}

	// Filter by ownership
	db = db.Where("user_id = ?", userID)

	// Apply status filters
	now := time.Now()
	switch q.Status {
	case "active":
		db = db.Where("is_active = ? AND (expires_at IS NULL OR expires_at > ?)", true, now)
	case "inactive":
		db = db.Where("is_active = ?", false)
	case "expired":
		db = db.Where("expires_at IS NOT NULL AND expires_at <= ?", now)
	}

	// Apply search parameters using case-insensitive ILIKE
	if q.Search != "" {
		searchTerm := "%" + q.Search + "%"
		db = db.Where("title ILIKE ? OR original_url ILIKE ? OR short_code ILIKE ?", searchTerm, searchTerm, searchTerm)
	}

	// Get total matching count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Validate and apply whitelisted sort parameters
	sortField := "created_at"
	switch q.Sort {
	case "created_at", "updated_at", "expires_at":
		sortField = q.Sort
	}

	sortOrder := "desc"
	if q.Order == "asc" {
		sortOrder = "asc"
	}

	db = db.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// Apply pagination limits
	offset := (q.Page - 1) * q.Limit
	if err := db.Offset(offset).Limit(q.Limit).Find(&links).Error; err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

// Update persists changes to a link's mutable parameters.
func (r *linkRepository) Update(ctx context.Context, link *models.Link) error {
	return r.db.WithContext(ctx).Save(link).Error
}

// SoftDelete soft-deletes a link from active query ranges using GORM conventions.
func (r *linkRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Link{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainErrors.ErrNotFound
	}
	return nil
}

// ExistsAlias checks if a specific short code or alias is already stored.
func (r *linkRepository) ExistsAlias(ctx context.Context, alias string) (bool, error) {
	var count int64
	// Unscoped search ensures reserved custom aliases cannot conflict with soft-deleted items.
	err := r.db.WithContext(ctx).Unscoped().Model(&models.Link{}).Where("short_code = ?", alias).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
