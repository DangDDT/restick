package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const dbFileName = "restick.sqlite"

// Open opens the durable factory database in dataDir, applies the connection
// pragmas and pending migrations. There is no in-memory fallback: an unwritable
// or unusable data dir fails startup.
func Open(ctx context.Context, dataDir string) (*sql.DB, error) {
	dir, err := prepareDataDir(dataDir)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dsn(filepath.Join(dir, dbFileName)))
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", filepath.Join(dir, dbFileName), err)
	}
	// One connection serializes short write transactions; never hold one while
	// running the Python child or streaming a ZIP.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: connect %s: %w", filepath.Join(dir, dbFileName), err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(ctx, db, migrations); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func dsn(dbPath string) string {
	u := url.URL{Scheme: "file", Path: dbPath}
	return u.String() +
		"?_pragma=foreign_keys(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(FULL)" +
		"&_pragma=busy_timeout(5000)"
}
