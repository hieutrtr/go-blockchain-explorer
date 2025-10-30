package db

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes database migrations from the migrations directory
// It applies all pending up migrations to bring the database schema to the latest version
func RunMigrations(config *Config, migrationsPath string, logger *slog.Logger) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}
	if migrationsPath == "" {
		return fmt.Errorf("migrationsPath cannot be empty")
	}

	logger.Info("starting database migrations",
		slog.String("migrations_path", migrationsPath),
		slog.String("database", config.Name))

	// Build connection string for migrate library
	// migrate uses a slightly different format with additional parameters
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Name,
	)

	// Create migrate instance
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connString,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("database schema is up to date, no migrations needed")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		logger.Warn("failed to get migration version after successful migration", slog.Any("error", err))
	} else {
		logger.Info("migrations completed successfully",
			slog.Uint64("version", uint64(version)),
			slog.Bool("dirty", dirty))
	}

	return nil
}

// RollbackMigrations rolls back the last migration
// This should be used with caution, typically only in development or disaster recovery
func RollbackMigrations(config *Config, migrationsPath string, logger *slog.Logger) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}
	if migrationsPath == "" {
		return fmt.Errorf("migrationsPath cannot be empty")
	}

	logger.Warn("rolling back database migrations",
		slog.String("migrations_path", migrationsPath),
		slog.String("database", config.Name))

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Name,
	)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connString,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Roll back one migration
	if err := m.Steps(-1); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("no migrations to roll back")
			return nil
		}
		return fmt.Errorf("failed to roll back migration: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		logger.Warn("failed to get migration version after rollback", slog.Any("error", err))
	} else {
		logger.Info("migration rolled back successfully",
			slog.Uint64("version", uint64(version)),
			slog.Bool("dirty", dirty))
	}

	return nil
}

// GetMigrationVersion returns the current migration version
func GetMigrationVersion(config *Config, migrationsPath string) (uint, bool, error) {
	if config == nil {
		return 0, false, fmt.Errorf("config cannot be nil")
	}
	if migrationsPath == "" {
		return 0, false, fmt.Errorf("migrationsPath cannot be empty")
	}

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Name,
	)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connString,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}
