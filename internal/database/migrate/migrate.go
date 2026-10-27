// Package migrate wraps golang-migrate for schema migrations.
package migrate

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator handles database migrations
type Migrator struct {
	m *migrate.Migrate
}

// Config holds migration configuration
type Config struct {
	DatabaseURL    string
	MigrationsPath string
}

// New creates a new Migrator instance
func New(cfg Config) (*Migrator, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", cfg.MigrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{m: m}, nil
}

// Up runs all pending migrations
func (mg *Migrator) Up() error {
	if err := mg.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

// Down rolls back all migrations
func (mg *Migrator) Down() error {
	if err := mg.m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

// Steps runs n migrations (positive = up, negative = down)
func (mg *Migrator) Steps(n int) error {
	if err := mg.m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration steps failed: %w", err)
	}
	return nil
}

// Rollback rolls back the last migration
func (mg *Migrator) Rollback() error {
	return mg.Steps(-1)
}

// Version returns the current migration version
func (mg *Migrator) Version() (uint, bool, error) {
	return mg.m.Version()
}

// Force sets the migration version without running migrations
// Use with caution - this is for fixing dirty migrations
func (mg *Migrator) Force(version int) error {
	return mg.m.Force(version)
}

// Close closes the migrator
func (mg *Migrator) Close() error {
	sourceErr, dbErr := mg.m.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return dbErr
}
