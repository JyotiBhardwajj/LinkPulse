// Package repository provides interfaces and implementations for database operations.
package repository

import (
	"context"

	"gorm.io/gorm"
)

// TransactionManager coordinates database transaction bounds with transaction-scoped RepositoryManager injection.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(txRepo RepositoryManager) error) error
}

type transactionManager struct {
	db *gorm.DB
}

// NewTransactionManager creates a unified TransactionManager.
func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &transactionManager{db: db}
}

// WithinTransaction runs operations inside a database transaction with a transaction-scoped RepositoryManager.
func (m *transactionManager) WithinTransaction(ctx context.Context, fn func(txRepo RepositoryManager) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepoMgr := NewRepositoryManager(tx)
		return fn(txRepoMgr)
	})
}
