package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

// legacyDatabase builds a real schema-2 data dir from the two migration files
// that shipped before character reference lists existed.
func legacyDatabase(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn(filepath.Join(dir, DBFileName)))
	if err != nil {
		t.Fatalf("open schema 2 database: %v", err)
	}
	db.SetMaxOpenConns(1)

	all, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	legacy := make([]migration, 0, 2)
	for _, step := range all {
		if step.version <= 2 {
			legacy = append(legacy, step)
		}
	}
	if len(legacy) != 2 {
		t.Fatalf("legacy migrations = %d, want 2", len(legacy))
	}
	if err := migrate(context.Background(), db, legacy); err != nil {
		db.Close()
		t.Fatalf("apply legacy migrations: %v", err)
	}
	return db
}

func characterReferenceIDs(t *testing.T, db *sql.DB, characterID string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT asset_id FROM character_reference_assets
		WHERE character_id = ? ORDER BY position`, characterID)
	if err != nil {
		t.Fatalf("read character references: %v", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan character reference: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read character references: %v", err)
	}
	return ids
}

func TestCharacterReferencesMigrationPreservesData(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	draft := `{"schema_version":1,"rows":1,"character_id":"char-ref","free_text":"giữ nguyên tiếng Việt"}`
	snapshot := `{"schema_version":1,"cells":[],"character":{"id":"char-ref","name":"Sentinel"}}`
	prompt := "SENTINEL PROMPT — giữ nguyên"

	db := legacyDatabase(t, dir)
	execSQL(t, db, `INSERT INTO reference_assets (id, filename, content_type, byte_size, width, height, checksum_sha256)
		VALUES ('ref-a', 'a.png', 'image/png', 10, 4, 4, ?)`, checksum)
	execSQL(t, db, `INSERT INTO characters (id, name, reference_asset_id) VALUES ('char-ref', 'Ref', 'ref-a')`)
	execSQL(t, db, `INSERT INTO characters (id, name) VALUES ('char-null', 'No ref')`)
	execSQL(t, db, `INSERT INTO designs (id, name, character_id, rows, columns, outline_width, outline_color,
		draft_json, snapshot_json, compiled_prompt)
		VALUES ('design-ref', 'Sentinel', 'char-ref', 1, 1, 4, '#FFFFFF', ?, ?, ?)`, draft, snapshot, prompt)
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	reopened, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("upgrade from schema 2: %v", err)
	}

	if got := characterReferenceIDs(t, reopened, "char-ref"); !reflect.DeepEqual(got, []string{"ref-a"}) {
		t.Fatalf("char-ref references = %v, want [ref-a]", got)
	}
	if got := characterReferenceIDs(t, reopened, "char-null"); len(got) != 0 {
		t.Fatalf("char-null references = %v, want none", got)
	}

	var gotDraft, gotSnapshot, gotPrompt string
	if err := reopened.QueryRow(`SELECT draft_json, snapshot_json, compiled_prompt FROM designs WHERE id = 'design-ref'`).
		Scan(&gotDraft, &gotSnapshot, &gotPrompt); err != nil {
		t.Fatalf("read sentinel design: %v", err)
	}
	if gotDraft != draft || gotSnapshot != snapshot || gotPrompt != prompt {
		t.Fatalf("migration rewrote design content: %q / %q / %q", gotDraft, gotSnapshot, gotPrompt)
	}

	var assetRows int
	if err := reopened.QueryRow(`SELECT COUNT(*) FROM reference_assets WHERE id = 'ref-a'`).Scan(&assetRows); err != nil || assetRows != 1 {
		t.Fatalf("reference asset row = %d (%v)", assetRows, err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("close upgraded database: %v", err)
	}

	again, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("reopen upgraded database: %v", err)
	}
	defer again.Close()

	if got := characterReferenceIDs(t, again, "char-ref"); !reflect.DeepEqual(got, []string{"ref-a"}) {
		t.Fatalf("references after reopen = %v, want [ref-a] without duplicates", got)
	}
	var violations int
	fkRows, err := again.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	for fkRows.Next() {
		violations++
	}
	if err := fkRows.Err(); err != nil {
		fkRows.Close()
		t.Fatalf("foreign_key_check: %v", err)
	}
	fkRows.Close()
	if violations != 0 {
		t.Fatalf("foreign key violations = %d", violations)
	}
	var singularColumns int
	if err := again.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('characters') WHERE name = 'reference_asset_id'`).
		Scan(&singularColumns); err != nil {
		t.Fatalf("inspect characters columns: %v", err)
	}
	if singularColumns != 0 {
		t.Fatal("characters.reference_asset_id still exists after migration")
	}
}
