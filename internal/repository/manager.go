// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"gorm.io/gorm"
)

// RepositoryManager coordinates the lifecycle and instantiation of database repositories.
type RepositoryManager interface {
	Users() UserRepository
	Links() LinkRepository
	Analytics() AnalyticsRepository
}

type repositoryManager struct {
	userRepo      UserRepository
	linkRepo      LinkRepository
	analyticsRepo AnalyticsRepository
}

// NewRepositoryManager creates a unified RepositoryManager.
func NewRepositoryManager(db *gorm.DB) RepositoryManager {
	return &repositoryManager{
		userRepo:      NewUserRepository(db),
		linkRepo:      NewLinkRepository(db),
		analyticsRepo: NewAnalyticsRepository(db),
	}
}

// Users returns the UserRepository implementation.
func (m *repositoryManager) Users() UserRepository {
	return m.userRepo
}

// Links returns the LinkRepository implementation.
func (m *repositoryManager) Links() LinkRepository {
	return m.linkRepo
}

// Analytics returns the AnalyticsRepository implementation.
func (m *repositoryManager) Analytics() AnalyticsRepository {
	return m.analyticsRepo
}
