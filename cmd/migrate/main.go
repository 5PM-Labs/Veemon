// Command migrate is a CLI for database migrations and seeding.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/5PM-Labs/veemon/internal/config"
	"github.com/5PM-Labs/veemon/internal/database/migrate"
	"github.com/5PM-Labs/veemon/internal/database/seeds"
	"github.com/5PM-Labs/veemon/internal/infra/database"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Define commands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Build database URL
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	// Get migrations path
	migrationsPath := getMigrationsPath()

	switch command {
	case "up", "migrate":
		runMigrate(dbURL, migrationsPath)
	case "down":
		runDown(dbURL, migrationsPath)
	case "rollback":
		runRollback(dbURL, migrationsPath)
	case "status", "version":
		runStatus(dbURL, migrationsPath)
	case "force":
		if len(os.Args) < 3 {
			fmt.Println("Usage: migrate force <version>")
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Invalid version: %s\n", os.Args[2])
			os.Exit(1)
		}
		runForce(dbURL, migrationsPath, version)
	case "create":
		if len(os.Args) < 3 {
			fmt.Println("Usage: migrate create <name>")
			os.Exit(1)
		}
		runCreate(migrationsPath, os.Args[2])
	case "seed":
		runSeed(cfg)
	case "fresh":
		runFresh(dbURL, migrationsPath, cfg)
	case "refresh":
		runRefresh(dbURL, migrationsPath, cfg)
	case "reset":
		runReset(dbURL, migrationsPath)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Database Migration Tool

Usage:
  migrate <command> [arguments]

Commands:
  up, migrate     Run all pending migrations
  down            Rollback all migrations
  rollback        Rollback the last migration
  status          Show current migration version
  force <version> Force set migration version (use with caution)
  create <name>   Create a new migration file
  seed            Run database seeders
  fresh           Drop all tables and re-run all migrations
  refresh         Rollback all migrations and re-run them
  reset           Rollback all migrations

Examples:
  migrate up
  migrate rollback
  migrate create add_users_table
  migrate seed
  migrate fresh
  migrate force 1`)
}

func getMigrationsPath() string {
	// Try to find migrations directory
	paths := []string{
		"migrations",
		"./migrations",
		"../migrations",
		"../../migrations",
	}

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Default to current directory + migrations
	absPath, _ := filepath.Abs("migrations")
	return absPath
}

func runMigrate(dbURL, migrationsPath string) {
	fmt.Println("Running migrations...")

	m, err := migrate.New(migrate.Config{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
	})
	if err != nil {
		fmt.Printf("Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = m.Close() }()

	if err := m.Up(); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		os.Exit(1)
	}

	version, dirty, _ := m.Version()
	fmt.Printf("Migrations complete. Current version: %d (dirty: %v)\n", version, dirty)
}

func runDown(dbURL, migrationsPath string) {
	fmt.Println("Rolling back all migrations...")

	m, err := migrate.New(migrate.Config{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
	})
	if err != nil {
		fmt.Printf("Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = m.Close() }()

	if err := m.Down(); err != nil {
		fmt.Printf("Rollback failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("All migrations rolled back successfully.")
}

func runRollback(dbURL, migrationsPath string) {
	fmt.Println("Rolling back last migration...")

	m, err := migrate.New(migrate.Config{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
	})
	if err != nil {
		fmt.Printf("Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = m.Close() }()

	if err := m.Rollback(); err != nil {
		fmt.Printf("Rollback failed: %v\n", err)
		os.Exit(1)
	}

	version, dirty, _ := m.Version()
	fmt.Printf("Rollback complete. Current version: %d (dirty: %v)\n", version, dirty)
}

func runStatus(dbURL, migrationsPath string) {
	m, err := migrate.New(migrate.Config{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
	})
	if err != nil {
		fmt.Printf("Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = m.Close() }()

	version, dirty, err := m.Version()
	if err != nil {
		fmt.Printf("No migrations have been run yet.\n")
		return
	}

	fmt.Printf("Current migration version: %d\n", version)
	if dirty {
		fmt.Println("WARNING: Database is in dirty state. You may need to fix this manually.")
	}
}

func runForce(dbURL, migrationsPath string, version int) {
	fmt.Printf("Forcing migration version to %d...\n", version)

	m, err := migrate.New(migrate.Config{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
	})
	if err != nil {
		fmt.Printf("Failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = m.Close() }()

	if err := m.Force(version); err != nil {
		fmt.Printf("Force failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Migration version forced to %d\n", version)
}

func runCreate(migrationsPath, name string) {
	timestamp := time.Now().Format("20060102150405")

	// Get the next migration number
	files, err := os.ReadDir(migrationsPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("Failed to read migrations directory: %v\n", err)
		os.Exit(1)
	}

	maxNum := 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		var num int
		if _, err := fmt.Sscanf(f.Name(), "%06d_", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}

	nextNum := maxNum + 1

	// Create migration files
	upFile := filepath.Join(migrationsPath, fmt.Sprintf("%06d_%s.up.sql", nextNum, name))
	downFile := filepath.Join(migrationsPath, fmt.Sprintf("%06d_%s.down.sql", nextNum, name))

	upContent := fmt.Sprintf("-- %06d_%s.up.sql\n-- Created at: %s\n\n-- Add your migration SQL here\n", nextNum, name, timestamp)
	downContent := fmt.Sprintf("-- %06d_%s.down.sql\n-- Created at: %s\n\n-- Add your rollback SQL here\n", nextNum, name, timestamp)

	if err := os.WriteFile(upFile, []byte(upContent), 0600); err != nil {
		fmt.Printf("Failed to create up migration: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(downFile, []byte(downContent), 0600); err != nil {
		fmt.Printf("Failed to create down migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created migration files:\n  %s\n  %s\n", upFile, downFile)
}

func runSeed(cfg *config.Config) {
	fmt.Println("Running seeders...")

	dbCfg := database.Config{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser, Password: cfg.DBPassword,
		Name: cfg.DBName, SSLMode: cfg.DBSSLMode, Timezone: cfg.DBTimezone,
	}
	db, err := pgxpool.New(context.Background(), dbCfg.WithPassword(dbCfg.DSN()))
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	seeder := seeds.New(db)
	if err := seeder.SeedAll(ctx); err != nil {
		fmt.Printf("Seeding failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seeding complete!")
}

func runFresh(dbURL, migrationsPath string, cfg *config.Config) {
	fmt.Println("Running fresh migration (drop all and migrate)...")

	// First, rollback all migrations
	m, err := migrate.New(migrate.Config{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
	})
	if err != nil {
		fmt.Printf("Failed to create migrator: %v\n", err)
		os.Exit(1)
	}

	_ = m.Down() // Ignore error if no migrations
	_ = m.Close()

	// Then run migrations
	runMigrate(dbURL, migrationsPath)

	// Ask if user wants to seed
	var seedFlag bool
	flag.BoolVar(&seedFlag, "seed", false, "Run seeders after migration")

	// Check if --seed flag is present in remaining args
	for _, arg := range os.Args[2:] {
		if arg == "--seed" || arg == "-seed" {
			seedFlag = true
			break
		}
	}

	if seedFlag {
		runSeed(cfg)
	}
}

func runRefresh(dbURL, migrationsPath string, cfg *config.Config) {
	fmt.Println("Refreshing migrations (rollback and migrate)...")

	runReset(dbURL, migrationsPath)
	runMigrate(dbURL, migrationsPath)

	// Check if --seed flag is present
	for _, arg := range os.Args[2:] {
		if arg == "--seed" || arg == "-seed" {
			runSeed(cfg)
			break
		}
	}
}

func runReset(dbURL, migrationsPath string) {
	fmt.Println("Resetting database (rollback all)...")
	runDown(dbURL, migrationsPath)
}
