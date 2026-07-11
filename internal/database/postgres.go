// Package database manages connection pools and health checks for PostgreSQL.
package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"linkpulse/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresDB manages the PostgreSQL database connection pool.
type PostgresDB struct {
	DB    *gorm.DB
	sqlDB *sql.DB
}

// NewPostgresDB establishes a connection pool to PostgreSQL using GORM.
func NewPostgresDB(cfg config.DatabaseConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true, // Performance optimization for simple writes
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}

	// Set connection pool limits
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Validate connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Successfully connected to PostgreSQL database")

	return &PostgresDB{
		DB:    db,
		sqlDB: sqlDB,
	}, nil
}

// Ping verifies the database connection remains active.
func (p *PostgresDB) Ping() error {
	if p.sqlDB == nil {
		return fmt.Errorf("database connection not initialized")
	}
	return p.sqlDB.Ping()
}

// Close gracefully releases the database connection pool resources.
func (p *PostgresDB) Close() error {
	if p.sqlDB == nil {
		return nil
	}
	slog.Info("Closing PostgreSQL connection pool")
	return p.sqlDB.Close()
}
