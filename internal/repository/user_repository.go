// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"
	"errors"

	domainErrors "linkpulse/internal/errors"
	"linkpulse/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines the database operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository returns a new instance of UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create persists a user in the database.
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domainErrors.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// FindByID retrieves a user by their UUID.
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email address.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Update saves modifications to a user record and asserts exactly one row was affected.
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	db := r.db.WithContext(ctx).Save(user)
	if db.Error != nil {
		return db.Error
	}
	if db.RowsAffected != 1 {
		return domainErrors.ErrNotFound
	}
	return nil
}
