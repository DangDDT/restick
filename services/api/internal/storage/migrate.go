package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migration is one embedded schema step; version is the numeric filename prefix.
type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("storage: read migrations: %w", err)
	}
	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		file := entry.Name()
		prefix, rest, ok := strings.Cut(strings.TrimSuffix(file, ".sql"), "_")
		if !ok {
			return nil, fmt.Errorf("storage: migration %q must be named <version>_<name>.sql", file)
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version < 1 {
			return nil, fmt.Errorf("storage: migration %q has invalid version prefix", file)
		}
		body, err := migrationFiles.ReadFile(path.Join("migrations", file))
		if err != nil {
			return nil, fmt.Errorf("storage: read migration %q: %w", file, err)
		}
		migrations = append(migrations, migration{version: version, name: rest, sql: string(body)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	for i, m := range migrations {
		if m.version != i+1 {
			return nil, fmt.Errorf("storage: migrations must be numbered from 1 without gaps, got %d at position %d", m.version, i+1)
		}
	}
	return migrations, nil
}

// migrate applies every migration newer than the recorded schema version. Each
// migration and its version record commit in one transaction.
func migrate(ctx context.Context, db *sql.DB, migrations []migration) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	)`)
	if err != nil {
		return fmt.Errorf("storage: prepare schema_migrations: %w", err)
	}

	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("storage: read schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if m.version != current+1 {
			return fmt.Errorf("storage: migration %d cannot follow schema version %d", m.version, current)
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return fmt.Errorf("storage: migration %04d_%s: %w", m.version, m.name, err)
		}
		current = m.version
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
		return err
	}
	return tx.Commit()
}
