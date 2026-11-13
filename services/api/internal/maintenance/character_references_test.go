package maintenance

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/storage"
)

// buildLegacyReferenceFixture writes a schema-2 data dir from the two real
// migration files: one character with a singular reference image and its bytes
// under sources/<asset>/original.png.
func buildLegacyReferenceFixture(t *testing.T) (dir, assetID string, image []byte) {
	t.Helper()
	ctx := context.Background()
	dir = t.TempDir()
	u := url.URL{Scheme: "file", Path: filepath.Join(dir, storage.DBFileName)}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		t.Fatalf("open schema 2 database: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	execSQL(t, db, `CREATE TABLE schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	)`)
	for version, name := range []string{"0001_factory.sql", "0002_design_sheets.sql"} {
		body, err := os.ReadFile(filepath.Join("..", "storage", "migrations", name))
		if err != nil {
			t.Fatalf("read legacy migration %s: %v", name, err)
		}
		execSQL(t, db, string(body))
		execSQL(t, db, `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, version+1, name)
	}

	store, err := assets.NewStore(dir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	source, err := store.SaveSource(ctx, bytes.NewReader(encodePNG(t)), "reference.png")
	if err != nil {
		t.Fatalf("SaveSource: %v", err)
	}
	execSQL(t, db, `INSERT INTO reference_assets (id, filename, content_type, byte_size, width, height, checksum_sha256)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		source.ID, source.Filename, source.MediaType, source.ByteSize, source.Width, source.Height, source.ChecksumSHA256)
	execSQL(t, db, `INSERT INTO characters (id, name, reference_asset_id) VALUES ('char-legacy', 'Legacy ref', ?)`, source.ID)

	stored, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(source.RelativePath)))
	if err != nil {
		t.Fatalf("read stored reference: %v", err)
	}
	return dir, source.ID, stored
}

func TestRestoreLegacyCharacterReferences(t *testing.T) {
	ctx := context.Background()
	dir, assetID, image := buildLegacyReferenceFixture(t)
	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := Backup(ctx, dir, backupDir); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	target := filepath.Join(t.TempDir(), "data")
	if err := Restore(ctx, backupDir, target, writePassword(t, newPassword)); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	u := url.URL{Scheme: "file", Path: filepath.Join(target, storage.DBFileName)}
	db, err := sql.Open("sqlite", u.String()+"?mode=ro")
	if err != nil {
		t.Fatalf("open restored database: %v", err)
	}
	defer db.Close()

	var version int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read restored schema version: %v", err)
	}
	if version != 4 {
		t.Fatalf("restored schema version = %d, want 4", version)
	}
	rows, err := db.Query(`SELECT asset_id, position FROM character_reference_assets WHERE character_id = 'char-legacy'`)
	if err != nil {
		t.Fatalf("read restored character references: %v", err)
	}
	defer rows.Close()
	type association struct {
		assetID  string
		position int
	}
	found := []association{}
	for rows.Next() {
		var item association
		if err := rows.Scan(&item.assetID, &item.position); err != nil {
			t.Fatalf("scan restored reference: %v", err)
		}
		found = append(found, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read restored character references: %v", err)
	}
	if len(found) != 1 || found[0].assetID != assetID || found[0].position != 0 {
		t.Fatalf("restored references = %+v, want %s at position 0", found, assetID)
	}

	restored, err := os.ReadFile(filepath.Join(target, "sources", assetID, "original.png"))
	if err != nil {
		t.Fatalf("read restored reference image: %v", err)
	}
	if !bytes.Equal(restored, image) {
		t.Fatal("restored reference bytes differ from the backup")
	}
}

func TestRestoreRejectsMissingCharacterReferences(t *testing.T) {
	f := buildFixture(t)
	backupDir := backupFixture(t, f)

	dbPath := filepath.Join(backupDir, storage.DBFileName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open backup database: %v", err)
	}
	// RED on the old schema: the table does not exist yet, so nothing is dropped.
	execSQL(t, db, `DROP TABLE IF EXISTS character_reference_assets`)
	if err := db.Close(); err != nil {
		t.Fatalf("close backup database: %v", err)
	}
	digest, size := digestOf(t, dbPath)
	editManifest(t, backupDir, func(m *backupManifest) {
		setEntry(m, storage.DBFileName, digest, size)
	})

	target := filepath.Join(t.TempDir(), "data")
	err = Restore(context.Background(), backupDir, target, writePassword(t, newPassword))
	if !errors.Is(err, ErrBackupInvalid) {
		t.Fatalf("Restore error = %v, want ErrBackupInvalid", err)
	}
	assertOnlyLock(t, target)
}
