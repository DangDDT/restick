package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const checksum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func openTemp(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open(%s): %v", dir, err)
	}
	t.Cleanup(func() { db.Close() })
	return db, dir
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func seedFactoryRecords(t *testing.T, db *sql.DB) {
	t.Helper()
	execSQL(t, db, `INSERT INTO owner (id, username, password_hash) VALUES (1, 'admin', 'bcrypt-hash')`)
	execSQL(t, db, `INSERT INTO fragment_groups (id, name) VALUES ('g-visual', 'Thị giác')`)
	execSQL(t, db, `INSERT INTO fragments (id, group_id, name, prompt, scope) VALUES ('v-d01-001', 'g-visual', 'Mèo', 'A cheerful cat', 'identity')`)
	execSQL(t, db, `INSERT INTO reference_assets (id, filename, content_type, byte_size, width, height, checksum_sha256) VALUES ('ref-1', 'cat.png', 'image/png', 2048, 512, 512, ?)`, checksum)
	execSQL(t, db, `INSERT INTO characters (id, name) VALUES ('char-1', 'Mèo Mun')`)
	execSQL(t, db, `INSERT INTO character_reference_assets (character_id, asset_id, position) VALUES ('char-1', 'ref-1', 0)`)
	execSQL(t, db, `INSERT INTO designs (id, name, character_id, rows, columns, outline_width, outline_color, draft_json, snapshot_json, compiled_prompt) VALUES ('design-1', 'Tết', 'char-1', 2, 2, 4, '#FFFFFF', '{"schema_version":1,"rows":2}', '{"schema_version":1,"cells":[]}', 'A cat celebrating Tet')`)
	execSQL(t, db, `INSERT INTO packs (id, slug, name, author, design_id, status, current_version) VALUES ('pack-1', 'tet-cat', 'Tết Cat', 'Restick', 'design-1', 'ready', 1)`)
	execSQL(t, db, `INSERT INTO pack_tags (pack_id, tag) VALUES ('pack-1', 'tet')`)
	execSQL(t, db, `INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES ('job-1', 'pack-1', 1, 'succeeded', '{"schema_version":1,"rows":2,"columns":2}')`)
	execSQL(t, db, `INSERT INTO pack_versions (pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir) VALUES ('pack-1', 1, 'job-1', ?, 4, '{"schema":"restick.pack.v1"}', 'bundles/pack-1/v1.zip', 4096, 'packs/pack-1/v1/thumb.webp', 'packs/pack-1/v1')`, checksum)
	execSQL(t, db, `INSERT INTO stickers (pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height) VALUES ('pack-1', 1, 0, '000.webp', 'packs/pack-1/v1/000.webp', 'packs/pack-1/v1/thumbs/thumb_000.webp', ?, 512, 512)`, checksum)
	execSQL(t, db, `INSERT INTO api_keys (id, name, prefix, key_hash) VALUES ('key-1', 'phone', 'rsk_abcd', x'01020304')`)
	execSQL(t, db, `INSERT INTO pack_tombstones (pack_id, version, revoked_at, reason_code, checksum_sha256) VALUES ('pack-1', 1, '2026-11-10T00:00:00Z', 'author_request', ?)`, checksum)
	execSQL(t, db, `INSERT INTO seed_versions (name, version) VALUES ('restick.seed.v1', 1)`)
}

func TestReopenPreservesRecords(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	db, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	seedFactoryRecords(t, db)
	var createdAt string
	if err := db.QueryRow(`SELECT created_at FROM packs WHERE id = 'pack-1'`).Scan(&createdAt); err != nil {
		t.Fatalf("read created_at: %v", err)
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err != nil {
		t.Fatalf("packs.created_at %q is not a UTC second timestamp: %v", createdAt, err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, DBFileName)); err != nil {
		t.Fatalf("database file missing after close: %v", err)
	}

	reopened, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer reopened.Close()

	var prompt, slug, status string
	var currentVersion int
	if err := reopened.QueryRow(`SELECT prompt FROM fragments WHERE id = 'v-d01-001'`).Scan(&prompt); err != nil {
		t.Fatalf("fragment lost: %v", err)
	}
	if prompt != "A cheerful cat" {
		t.Fatalf("fragment prompt = %q", prompt)
	}
	if err := reopened.QueryRow(`SELECT slug, status, current_version FROM packs WHERE id = 'pack-1'`).Scan(&slug, &status, &currentVersion); err != nil {
		t.Fatalf("pack lost: %v", err)
	}
	if slug != "tet-cat" || status != "ready" || currentVersion != 1 {
		t.Fatalf("pack = %q/%q/%d", slug, status, currentVersion)
	}
	var stickerChecksum string
	if err := reopened.QueryRow(`SELECT checksum_sha256 FROM stickers WHERE pack_id = 'pack-1' AND version = 1 AND sort_order = 0`).Scan(&stickerChecksum); err != nil {
		t.Fatalf("sticker lost: %v", err)
	}
	if stickerChecksum != checksum {
		t.Fatalf("sticker checksum = %q", stickerChecksum)
	}
	var keyHash []byte
	if err := reopened.QueryRow(`SELECT key_hash FROM api_keys WHERE id = 'key-1'`).Scan(&keyHash); err != nil {
		t.Fatalf("api key lost: %v", err)
	}
	if string(keyHash) != "\x01\x02\x03\x04" {
		t.Fatalf("api key hash = %x", keyHash)
	}
	var revokedAt string
	if err := reopened.QueryRow(`SELECT revoked_at FROM pack_tombstones WHERE pack_id = 'pack-1'`).Scan(&revokedAt); err != nil {
		t.Fatalf("tombstone lost: %v", err)
	}
	if revokedAt != "2026-11-10T00:00:00Z" {
		t.Fatalf("tombstone revoked_at = %q", revokedAt)
	}
}

func TestForeignKeysAndUniqueSlug(t *testing.T) {
	db, _ := openTemp(t)

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d (err %v)", foreignKeys, err)
	}

	execSQL(t, db, `INSERT INTO fragment_groups (id, name) VALUES ('g1', 'Group')`)
	execSQL(t, db, `INSERT INTO fragments (id, group_id, name, prompt, scope) VALUES ('f1', 'g1', 'Mèo', 'A cat', 'pack')`)
	if _, err := db.Exec(`INSERT INTO fragments (id, group_id, name, prompt, scope) VALUES ('f2', 'missing', 'Cáo', 'A fox', 'pack')`); err == nil {
		t.Fatal("fragment with unknown group was accepted")
	}
	if _, err := db.Exec(`DELETE FROM fragment_groups WHERE id = 'g1'`); err == nil {
		t.Fatal("group with members was deleted")
	}

	execSQL(t, db, `INSERT INTO packs (id, slug, name, author) VALUES ('p1', 'my-pack', 'Pack', 'Restick')`)
	if _, err := db.Exec(`INSERT INTO packs (id, slug, name, author) VALUES ('p2', 'My-Pack', 'Pack two', 'Restick')`); err == nil {
		t.Fatal("case-insensitive duplicate slug was accepted")
	}
	if _, err := db.Exec(`INSERT INTO stickers (pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height)
		VALUES ('p1', 9, 0, '000.webp', 'a', 'b', ?, 512, 512)`, checksum); err == nil {
		t.Fatal("sticker without a committed pack version was accepted")
	}
}

func TestOneActiveJobPerPack(t *testing.T) {
	db, _ := openTemp(t)
	execSQL(t, db, `INSERT INTO packs (id, slug, name, author) VALUES ('p1', 'pack', 'Pack', 'Restick')`)

	execSQL(t, db, `INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES ('job-1', 'p1', 1, 'queued', '{"schema_version":1}')`)
	if _, err := db.Exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES ('job-2', 'p1', 2, 'queued', '{"schema_version":1}')`); err == nil {
		t.Fatal("second active job for the same pack was accepted")
	}
	execSQL(t, db, `UPDATE processing_jobs SET status = 'failed' WHERE id = 'job-1'`)
	execSQL(t, db, `INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES ('job-3', 'p1', 2, 'queued', '{"schema_version":1}')`)

	// A second pack may run its own job.
	execSQL(t, db, `INSERT INTO packs (id, slug, name, author) VALUES ('p2', 'pack-two', 'Pack two', 'Restick')`)
	execSQL(t, db, `INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES ('job-4', 'p2', 1, 'queued', '{"schema_version":1}')`)
}

func TestMigrationFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", dsn(filepath.Join(t.TempDir(), "migrate.sqlite")))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	migrations := []migration{
		{version: 1, name: "good", sql: `CREATE TABLE good (id INTEGER PRIMARY KEY);`},
		{version: 2, name: "broken", sql: `CREATE TABLE broken (id INTEGER PRIMARY KEY);
			INSERT INTO missing_table (id) VALUES (1);`},
	}
	if err := migrate(ctx, db, migrations); err == nil {
		t.Fatal("broken migration was accepted")
	}

	var version int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != 1 {
		t.Fatalf("schema version = %d, want 1", version)
	}
	var name string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'good'`).Scan(&name); err != nil {
		t.Fatalf("committed migration table missing: %v", err)
	}
	err = db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'broken'`).Scan(&name)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed migration left table behind (err %v)", err)
	}

	// A retried run after the failure can still apply a corrected migration 2.
	migrations[1] = migration{version: 2, name: "fixed", sql: `CREATE TABLE fixed (id INTEGER PRIMARY KEY);`}
	if err := migrate(ctx, db, migrations); err != nil {
		t.Fatalf("retry after failed migration: %v", err)
	}
}

func TestAppLockRejectsSecondWriter(t *testing.T) {
	dir := t.TempDir()

	release, err := AcquireAppLock(dir)
	if err != nil {
		t.Fatalf("AcquireAppLock: %v", err)
	}
	if _, err := AcquireAppLock(dir); !errors.Is(err, ErrAppLocked) {
		t.Fatalf("second writer error = %v, want ErrAppLocked", err)
	}
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := release(); err != nil {
		t.Fatalf("second release: %v", err)
	}

	again, err := AcquireAppLock(dir)
	if err != nil {
		t.Fatalf("reacquire after release: %v", err)
	}
	if err := again(); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestAppLockRejectsUnusableLockFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores lock file permissions")
	}
	dir := t.TempDir()
	lockPath := filepath.Join(dir, LockFileName)
	if err := os.WriteFile(lockPath, nil, 0o400); err != nil {
		t.Fatalf("prepare read-only lock file: %v", err)
	}
	if _, err := AcquireAppLock(dir); err == nil || errors.Is(err, ErrAppLocked) {
		t.Fatalf("AcquireAppLock on read-only lock file = %v, want permission error", err)
	}
}

func TestOpenRejectsUnusableDataDir(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatalf("prepare file: %v", err)
	}
	db, err := Open(context.Background(), file)
	if err == nil || db != nil {
		t.Fatalf("Open on file path = (%v, %v), want error and no fallback", db, err)
	}
}

func TestOpenAppliesPragmas(t *testing.T) {
	db, _ := openTemp(t)

	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}
	var synchronous, foreignKeys, busyTimeout int
	if err := db.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatalf("read synchronous: %v", err)
	}
	if synchronous != 2 {
		t.Errorf("synchronous = %d, want 2 (FULL)", synchronous)
	}
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
	if stats := db.Stats(); stats.MaxOpenConnections != 1 {
		t.Errorf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestOpenHandlesURISpecialDataPath(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "data ?#dir")

	db, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open(%q): %v", dir, err)
	}
	execSQL(t, db, `INSERT INTO fragment_groups (id, name) VALUES ('g1', 'Group')`)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("reopen(%q): %v", dir, err)
	}
	defer reopened.Close()
	var name string
	if err := reopened.QueryRow(`SELECT name FROM fragment_groups WHERE id = 'g1'`).Scan(&name); err != nil {
		t.Fatalf("record lost through URI-escaped path: %v", err)
	}
	if name != "Group" {
		t.Fatalf("name = %q", name)
	}
}

func TestTimestampTextRejectsOffsetsAndFractions(t *testing.T) {
	db, _ := openTemp(t)
	execSQL(t, db, `INSERT INTO packs (id, slug, name, author) VALUES ('p1', 'pack', 'Pack', 'Restick')`)

	for _, bad := range []string{
		"2026-11-10T00:00:00+07:00",
		"2026-11-10T00:00:00.123Z",
		"2026-11-10 00:00:00Z",
	} {
		if _, err := db.Exec(`INSERT INTO pack_tombstones (pack_id, version, revoked_at, reason_code, checksum_sha256) VALUES ('p1', 1, ?, 'author_request', ?)`, bad, checksum); err == nil {
			t.Errorf("tombstone revoked_at %q was accepted", bad)
		}
	}
	execSQL(t, db, `INSERT INTO pack_tombstones (pack_id, version, revoked_at, reason_code, checksum_sha256) VALUES ('p1', 1, '2026-11-10T00:00:00Z', 'author_request', ?)`, checksum)

	for i, bad := range []string{"2026-11-10T00:00:00.5Z", "2026-11-10T00:00:00-03:00"} {
		if _, err := db.Exec(`INSERT INTO admin_sessions (id, token_hash, expires_at) VALUES (?, x'01', ?)`, "sess-"+string(rune('a'+i)), bad); err == nil {
			t.Errorf("session expires_at %q was accepted", bad)
		}
	}
	execSQL(t, db, `INSERT INTO admin_sessions (id, token_hash, expires_at) VALUES ('sess-ok', x'01', '2026-11-11T00:00:00Z')`)

	if _, err := db.Exec(`INSERT INTO api_keys (id, name, prefix, key_hash, expires_at) VALUES ('key-bad', 'k', 'rsk_', x'01', '2026-12-01T00:00:00+05:00')`); err == nil {
		t.Error("api key expires_at with offset was accepted")
	}
	execSQL(t, db, `INSERT INTO api_keys (id, name, prefix, key_hash) VALUES ('key-null', 'k', 'rsk_', x'01')`)
	execSQL(t, db, `INSERT INTO api_keys (id, name, prefix, key_hash, expires_at) VALUES ('key-ok', 'k', 'rsk_', x'02', '2026-12-01T00:00:00Z')`)
}

func TestMigration0002Schema(t *testing.T) {
	db, _ := openTemp(t)

	for _, table := range []string{"design_sheets", "api_key_packs", "api_key_pack_removals"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n); err != nil || n != 1 {
			t.Errorf("table %s missing (n=%d err=%v)", table, n, err)
		}
	}
	has := func(col string) bool {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM pragma_table_info('packs') WHERE name = ?`, col).Scan(&n); err != nil {
			t.Fatalf("table_info: %v", err)
		}
		return n == 1
	}
	if !has("sheet_id") {
		t.Error("packs.sheet_id missing")
	}
	for _, col := range []string{"source_id", "source_filename", "source_checksum_sha256", "source_width", "source_height", "source_byte_size"} {
		if has(col) {
			t.Errorf("packs.%s still present", col)
		}
	}

	execSQL(t, db, `INSERT INTO designs (id, name, rows, columns, outline_width, outline_color, draft_json, snapshot_json) VALUES ('d1', 'D', 2, 2, 4, '#FFFFFF', '{}', '{"schema_version":1}')`)
	const ins = `INSERT INTO design_sheets (id, design_id, provider, prompt_snapshot, filename, media_type, byte_size, width, height, checksum_sha256) VALUES (?, 'd1', ?, 'p', 's.png', 'image/png', 10, 64, 64, ?)`
	execSQL(t, db, ins, "s1", "chatgpt", checksum)
	if _, err := db.Exec(ins, "s2", "gemini", checksum); err == nil {
		t.Error("duplicate (design_id, checksum) accepted")
	}
	for _, bad := range []string{"", "Bad", "has space", "0123456789012345678901234567890123"} {
		if _, err := db.Exec(ins, "s-bad", bad, strings.Repeat("a", 64)); err == nil {
			t.Errorf("provider %q accepted", bad)
		}
	}
	if _, err := db.Exec(`DELETE FROM designs WHERE id = 'd1'`); err == nil {
		t.Error("deleting a design with a sheet succeeded")
	}
}

func TestMigration0004AppliesOnFreshDatabase(t *testing.T) {
	db, _ := openTemp(t)

	var version int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil || version != 4 {
		t.Fatalf("schema version = %d (err %v), want 4", version, err)
	}
	for _, col := range []string{"animated_path", "animated_checksum_sha256", "animated_byte_size", "animated_frames", "animated_duration_ms"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM pragma_table_info('stickers') WHERE name = ?`, col).Scan(&n); err != nil || n != 1 {
			t.Errorf("stickers.%s missing (n=%d err=%v)", col, n, err)
		}
	}

	execSQL(t, db, `INSERT INTO packs (id, slug, name, author) VALUES ('p1', 'pack', 'Pack', 'Restick')`)
	const ins = `INSERT INTO sticker_animations (id, pack_id, sticker_index, static_checksum_sha256, clip_id, clip_path, clip_byte_size, clip_checksum_sha256, status)
		VALUES (?, 'p1', ?, ?, 'c', 'clips/c.mp4', 10, ?, ?)`
	execSQL(t, db, ins, "a1", 0, checksum, checksum, "queued")
	var use int
	if err := db.QueryRow(`SELECT use_animated FROM sticker_animations WHERE id = 'a1'`).Scan(&use); err != nil || use != 1 {
		t.Errorf("use_animated default = %d (err %v), want 1", use, err)
	}
	if _, err := db.Exec(ins, "a2", 0, checksum, checksum, "queued"); err == nil {
		t.Error("second draft for the same sticker accepted")
	}
	if _, err := db.Exec(ins, "a3", 1, checksum, checksum, "done"); err == nil {
		t.Error("unknown status accepted")
	}
	if _, err := db.Exec(ins, "a4", -1, checksum, checksum, "queued"); err == nil {
		t.Error("negative sticker index accepted")
	}
	execSQL(t, db, `DELETE FROM packs WHERE id = 'p1'`)
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sticker_animations`).Scan(&n); err != nil || n != 0 {
		t.Errorf("drafts survived pack delete (n=%d err=%v)", n, err)
	}
}
