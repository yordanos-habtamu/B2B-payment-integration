package migrations

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	db        *pgxpool.Pool
	migrationDir string
}

type Migration struct {
	Version string
	Name    string
	SQL     string
}

func NewMigrator(db *pgxpool.Pool, migrationDir string) *Migrator {
	return &Migrator{
		db:        db,
		migrationDir: migrationDir,
	}
}

// Init creates the migrations table if it doesn't exist
func (m *Migrator) Init(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations(applied_at);
	`
	
	_, err := m.db.Exec(ctx, createTableSQL)
	return err
}

// LoadMigrations loads all migration files from the migration directory
func (m *Migrator) LoadMigrations() ([]*Migration, error) {
	files, err := ioutil.ReadDir(m.migrationDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []*Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Extract version from filename (e.g., "001_create_payments_table.sql")
		parts := strings.Split(file.Name(), "_")
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		name := strings.TrimSuffix(strings.Join(parts[1:], "_"), ".sql")

		// Read migration SQL
		filePath := filepath.Join(m.migrationDir, file.Name())
		sqlBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		migrations = append(migrations, &Migration{
			Version: version,
			Name:    name,
			SQL:     string(sqlBytes),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// GetAppliedMigrations returns a map of already applied migration versions
func (m *Migrator) GetAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return applied, nil
}

// Up applies all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	// Initialize migrations table
	if err := m.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	// Load all migration files
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			fmt.Printf("Migration %s already applied, skipping\n", migration.Version)
			continue
		}

		fmt.Printf("Applying migration %s: %s\n", migration.Version, migration.Name)
		
		// Start transaction
		tx, err := m.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %s: %w", migration.Version, err)
		}
		defer tx.Rollback(ctx)

		// Execute migration SQL
		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		// Record migration as applied
		if _, err := tx.Exec(ctx, 
			"INSERT INTO schema_migrations (version) VALUES ($1)", 
			migration.Version); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		fmt.Printf("Successfully applied migration %s\n", migration.Version)
	}

	fmt.Println("All migrations applied successfully")
	return nil
}

// Down rolls back the last migration (not implemented for safety)
func (m *Migrator) Down(ctx context.Context) error {
	return fmt.Errorf("rollback migrations not implemented for safety reasons")
}

// Status shows the current migration status
func (m *Migrator) Status(ctx context.Context) error {
	// Initialize migrations table
	if err := m.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	// Load all migration files
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	fmt.Println("Migration Status:")
	fmt.Println("=================")
	
	for _, migration := range migrations {
		status := "pending"
		if applied[migration.Version] {
			status = "applied"
		}
		fmt.Printf("%s %s: %s (%s)\n", migration.Version, migration.Name, status, migration.Version)
	}

	return nil
}

// CreateMigration creates a new migration file with the given name
func (m *Migrator) CreateMigration(name string) error {
	// Get next migration version
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	var nextVersion string
	if len(migrations) == 0 {
		nextVersion = "001"
	} else {
		lastVersion := migrations[len(migrations)-1].Version
		// Simple increment for demo - in production, you'd want more sophisticated versioning
		nextVersion = fmt.Sprintf("%03d", mustParseInt(lastVersion)+1)
	}

	// Create migration file
	filename := fmt.Sprintf("%s_%s.sql", nextVersion, name)
	filepath := filepath.Join(m.migrationDir, filename)

	template := fmt.Sprintf(`-- Migration: %s
-- Description: 

`, name)

	if err := ioutil.WriteFile(filepath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	fmt.Printf("Created migration file: %s\n", filename)
	return nil
}

func mustParseInt(s string) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		panic(err)
	}
	return result
}
