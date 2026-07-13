// Package database manages connection pools and health checks for PostgreSQL.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"linkpulse/internal/config"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // file source driver
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

// RunMigrations executes SQL migration files from the given directory path.
// It uses golang-migrate with the file:// source driver.
// ErrNoChange is silently ignored — all other errors fail startup immediately.
func (p *PostgresDB) RunMigrations(migrationsPath string) error {
	slog.Info("Running SQL migrations", "path", migrationsPath)

	driver, err := migratepgx.WithInstance(p.sqlDB, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("SQL migrations: no new migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	slog.Info("SQL migrations applied successfully", "version", version, "dirty", dirty)
	return nil
}

// Ping verifies the database connection remains active.
func (p *PostgresDB) Ping(ctx context.Context) error {
	if p.sqlDB == nil {
		return fmt.Errorf("database connection not initialized")
	}
	return p.sqlDB.PingContext(ctx)
}

// Ready satisfies the ReadinessChecker interface.
func (p *PostgresDB) Ready(ctx context.Context) error {
	return p.Ping(ctx)
}

// Close gracefully releases the database connection pool resources.
func (p *PostgresDB) Close() error {
	if p.sqlDB == nil {
		return nil
	}
	slog.Info("Closing PostgreSQL connection pool")
	return p.sqlDB.Close()
}
