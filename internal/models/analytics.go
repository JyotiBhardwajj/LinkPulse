// Package models defines the database schemas and DTO structs.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Analytics represents a click event tracked for a shortened URL.
type Analytics struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	LinkID    uuid.UUID `gorm:"type:uuid;not null;index" json:"link_id"`
	ClickedAt time.Time `gorm:"type:timestamp with time zone;not null;index" json:"clicked_at"`
	IPHash    string    `gorm:"type:varchar(64);not null" json:"ip_hash"`
	Country   string    `gorm:"type:varchar(100)" json:"country"`
	City      string    `gorm:"type:varchar(100)" json:"city"`
	Browser   string    `gorm:"type:varchar(100)" json:"browser"`
	OS        string    `gorm:"type:varchar(100)" json:"os"`
	Device    string    `gorm:"type:varchar(100)" json:"device"`
	Referrer  string    `gorm:"type:text" json:"referrer"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`
}
