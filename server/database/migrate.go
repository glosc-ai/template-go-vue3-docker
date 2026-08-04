package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed migrations/*/*.sql
var migrationFiles embed.FS

func Migrate(ctx context.Context, db *sql.DB, driver string) error {
	if driver != "postgres" && driver != "mysql" {
		return fmt.Errorf("migrating database: unsupported driver %q", driver)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	applied, err := appliedMigrations(ctx, db)
	if err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrationFiles, "migrations/"+driver)
	if err != nil {
		return fmt.Errorf("reading %s migrations: %w", driver, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if _, ok := applied[entry.Name()]; ok {
			continue
		}
		contents, err := migrationFiles.ReadFile("migrations/" + driver + "/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry.Name(), err)
		}
		if err := applyMigration(ctx, db, driver, entry.Name(), string(contents)); err != nil {
			return err
		}
	}
	return nil
}

func appliedMigrations(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("listing applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scanning migration version: %w", err)
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading migration versions: %w", err)
	}
	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, driver, version, contents string) error {
	if driver == "mysql" {
		for _, statement := range strings.Split(contents, "-- statement-breakpoint") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("applying migration %s: %w", version, err)
			}
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("recording migration %s: %w", version, err)
		}
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting migration %s: %w", version, err)
	}
	defer tx.Rollback()

	for _, statement := range strings.Split(contents, "-- statement-breakpoint") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("applying migration %s: %w", version, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("recording migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration %s: %w", version, err)
	}
	return nil
}
