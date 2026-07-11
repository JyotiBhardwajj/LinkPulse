// Package models defines the database schemas and DTO structs.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents the system user entity.
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Links        []Link         `gorm:"foreignKey:UserID" json:"links,omitempty"`
}
