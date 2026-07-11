// Package models defines the database schemas and DTO structs.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Link represents the shortened link entity.
type Link struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	OriginalURL string         `gorm:"type:text;not null" json:"original_url"`
	ShortCode   string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"short_code"`
	Title       string         `gorm:"type:varchar(255)" json:"title"`
	UserID      *uuid.UUID     `gorm:"type:uuid" json:"user_id,omitempty"`
	IsActive    bool           `gorm:"type:boolean;default:true;not null" json:"is_active"`
	ExpiresAt   *time.Time     `gorm:"type:timestamp with time zone" json:"expires_at,omitempty"`
	CreatedAt   time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
