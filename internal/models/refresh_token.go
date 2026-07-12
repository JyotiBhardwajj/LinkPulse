// Package models defines the database schemas and DTO structs.
package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents the session tracking token model.
type RefreshToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash  string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"token_hash"`
	DeviceName string     `gorm:"type:varchar(100)" json:"device_name"`
	IPHash     string     `gorm:"type:varchar(64);not null" json:"ip_hash"`
	UserAgent  string     `gorm:"type:text" json:"user_agent"`
	LastUsedAt time.Time  `gorm:"type:timestamp with time zone;not null" json:"last_used_at"`
	ExpiresAt  time.Time  `gorm:"type:timestamp with time zone;not null" json:"expires_at"`
	CreatedIP  string     `gorm:"type:varchar(100)" json:"created_ip"`
	CreatedAt  time.Time  `gorm:"type:timestamp with time zone;not null" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"type:timestamp with time zone;not null" json:"updated_at"`
	RevokedAt  *time.Time `gorm:"type:timestamp with time zone" json:"revoked_at"`
}
