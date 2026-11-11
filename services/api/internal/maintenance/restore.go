package maintenance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/storage"
)

// Restore rebuilds a data dir from a restick.backup.v1 folder produced by Backup.
//
// Restore is offline: it takes the app lock, and the target data dir must be
// empty apart from the .restick.lock this command itself creates. Existing data
// is never overwritten. Every file is copied into a staging directory inside the
// target (same filesystem) while its manifest digest is verified; the restored
// database is then opened, checked for schema, integrity, foreign keys and every
// file it references, and only a fully valid snapshot is renamed into place —
// database last, so a crash before that final rename never leaves a data dir that
// looks complete. The app lock is held across the finalization; the lock file is
// never renamed, so the lock cannot be bypassed.
//
// The required passwordFile supplies the new owner password. Restore replaces the
// owner hash with it, deletes every admin session and disables every restored API
// key — including keys that were already rotated away — before any server can
// read the data dir, so no credential from the backup survives.
//
// Checksums detect corruption and editing in transit; they do not authenticate
// the backup. See the package comment.
func Restore(ctx context.Context, inputDir, dataDir, passwordFile string) error {
	password, err := readPassword(passwordFile)
	if err != nil {
		return err
	}
	in, err := canonicalDir(inputDir)
	if err != nil {
		return err
	}
	dir, err := absolutePath(dataDir)
	if err != nil {
		return err
	}
	files, err := readManifest(in)
	if err != nil {
		return err
	}

	release, err := storage.AcquireAppLock(dir)
	if err != nil {
		return err
	}
	defer release()
	if err := ensureEmptyTarget(dir); err != nil {
		return err
	}

	staging, err := os.MkdirTemp(dir, ".restick-restore-")
	if err != nil {
		return fmt.Errorf("maintenance: create restore staging: %w", err)
	}
	defer os.RemoveAll(staging)

	if err := copyBackup(ctx, in, staging, files); err != nil {
		return err
	}
	if err := applyDatabase(ctx, staging, files, password); err != nil {
		return err
	}
	return publish(staging, dir)
}

// readPassword reads the new owner password before anything in the data dir is
// touched, so a missing secret never leaves a half-restored factory.
func readPassword(passwordFile string) ([]byte, error) {
	if strings.TrimSpace(passwordFile) == "" {
		return nil, ErrMissingPassword
	}
	body, err := os.ReadFile(passwordFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMissingPassword, err)
	}
	password := bytes.TrimSpace(body)
	if len(password) == 0 {
		return nil, ErrMissingPassword
	}
	return password, nil
}

// readManifest parses and validates the backup manifest. Every entry must be a
// unique path inside the backup allowlist (the database, sources/, packs/,
// bundles/); nothing else is ever read from the backup folder.
func readManifest(root string) (map[string]manifestFile, error) {
	body, err := os.ReadFile(filepath.Join(root, manifestFileName))
	if err != nil {
		return nil, fmt.Errorf("%w: read manifest: %v", ErrBackupInvalid, err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("%w: decode manifest: %v", ErrBackupInvalid, err)
	}
	if manifest.Schema != BackupSchema {
		return nil, fmt.Errorf("%w: manifest schema %q", ErrBackupInvalid, manifest.Schema)
	}
	files := make(map[string]manifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		rel, err := safeBackupPath(file.Path)
		if err != nil {
			return nil, err
		}
		if len(file.SHA256) != 64 || file.ByteSize < 0 {
			return nil, fmt.Errorf("%w: %s has an invalid digest", ErrBackupInvalid, rel)
		}
		if _, duplicate := files[rel]; duplicate {
			return nil, fmt.Errorf("%w: %s is listed twice", ErrBackupInvalid, rel)
		}
		file.Path = rel
		files[rel] = file
	}
	if _, ok := files[storage.DBFileName]; !ok {
		return nil, fmt.Errorf("%w: manifest lists no %s", ErrBackupInvalid, storage.DBFileName)
	}
	return files, nil
}

// safeBackupPath normalizes a manifest or database path and rejects anything
// that is absolute, escapes with .., or falls outside the backed-up subtrees.
func safeBackupPath(p string) (string, error) {
	norm := strings.TrimPrefix(strings.ReplaceAll(p, "\\", "/"), "./")
	clean := path.Clean(norm)
	if norm == "" || norm != clean || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, "../") || path.IsAbs(clean) || strings.ContainsRune(clean, 0) {
		return "", fmt.Errorf("%w: unsafe path %q", ErrBackupInvalid, p)
	}
	top, _, _ := strings.Cut(clean, "/")
	if top != storage.DBFileName && top != "sources" && top != "packs" && top != "bundles" {
		return "", fmt.Errorf("%w: %q is outside the backup allowlist", ErrBackupInvalid, p)
	}
	return clean, nil
}

// copyBackup copies every manifest file into staging, verifying the copied bytes
// against the manifest and refusing symlinks that leave the backup folder.
func copyBackup(ctx context.Context, srcRoot, dstRoot string, files map[string]manifestFile) error {
	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry := files[rel]
		src := filepath.Join(srcRoot, filepath.FromSlash(rel))
		resolved, err := filepath.EvalSymlinks(src)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrBackupInvalid, rel, err)
		}
		if !within(srcRoot, resolved) {
			return fmt.Errorf("%w: %s escapes the backup folder", ErrBackupInvalid, rel)
		}
		dst := filepath.Join(dstRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("maintenance: prepare %s: %w", rel, err)
		}
		digest, size, err := copyFile(ctx, src, dst)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrBackupInvalid, err)
		}
		if size != entry.ByteSize || digest != entry.SHA256 {
			return fmt.Errorf("%w: %s does not match its manifest digest", ErrBackupInvalid, rel)
		}
	}
	return nil
}

// applyDatabase validates the staged database, forces the new owner credentials
// and leaves a checkpointed database file behind, without WAL side files.
func applyDatabase(ctx context.Context, stagingDir string, files map[string]manifestFile, password []byte) error {
	db, err := storage.Open(ctx, stagingDir)
	if err != nil {
		return fmt.Errorf("%w: open restored database: %v", ErrBackupInvalid, err)
	}
	restoreErr := validateDatabase(ctx, db, files)
	if restoreErr == nil {
		restoreErr = resetCredentials(ctx, db, password)
	}
	if restoreErr == nil {
		if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			restoreErr = fmt.Errorf("maintenance: checkpoint restored database: %w", err)
		}
	}
	if closeErr := db.Close(); restoreErr == nil && closeErr != nil {
		restoreErr = fmt.Errorf("maintenance: close restored database: %w", closeErr)
	}
	if restoreErr != nil {
		return restoreErr
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(filepath.Join(stagingDir, storage.DBFileName+suffix)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("maintenance: remove %s: %w", storage.DBFileName+suffix, err)
		}
	}
	return nil
}

// requiredTables are the schema objects a restorable factory database must have.
var requiredTables = []string{
	"owner", "admin_sessions", "fragment_groups", "fragments", "reference_assets",
	"characters", "designs", "packs", "pack_tags", "pack_versions", "stickers",
	"processing_jobs", "api_keys", "pack_tombstones", "seed_versions", "schema_migrations",
}

func validateDatabase(ctx context.Context, db *sql.DB, files map[string]manifestFile) error {
	if err := checkSchema(ctx, db); err != nil {
		return err
	}
	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return fmt.Errorf("%w: integrity check: %v", ErrBackupInvalid, err)
	}
	if integrity != "ok" {
		return fmt.Errorf("%w: integrity check: %s", ErrBackupInvalid, integrity)
	}
	if err := checkForeignKeys(ctx, db); err != nil {
		return err
	}
	return checkReferencedFiles(ctx, db, files)
}

func checkSchema(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return fmt.Errorf("%w: read schema: %v", ErrBackupInvalid, err)
	}
	defer rows.Close()

	present := make(map[string]bool, len(requiredTables))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("%w: read schema: %v", ErrBackupInvalid, err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: read schema: %v", ErrBackupInvalid, err)
	}
	for _, table := range requiredTables {
		if !present[table] {
			return fmt.Errorf("%w: schema is missing table %s", ErrBackupInvalid, table)
		}
	}
	return nil
}

func checkForeignKeys(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("%w: foreign key check: %v", ErrBackupInvalid, err)
	}
	defer rows.Close()
	if rows.Next() {
		var table string
		if err := rows.Scan(&table, new(any), new(any), new(any)); err != nil {
			return fmt.Errorf("%w: foreign key check: %v", ErrBackupInvalid, err)
		}
		return fmt.Errorf("%w: foreign key violation in %s", ErrBackupInvalid, table)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: foreign key check: %v", ErrBackupInvalid, err)
	}
	return nil
}

// checkReferencedFiles proves every file the database points at is present in the
// backup with the size and digest the database recorded.
func checkReferencedFiles(ctx context.Context, db *sql.DB, files map[string]manifestFile) error {
	sources, err := sourceIndex(files)
	if err != nil {
		return err
	}
	if err := checkPackSources(ctx, db, sources); err != nil {
		return err
	}
	if err := checkReferenceAssets(ctx, db, sources); err != nil {
		return err
	}
	if err := checkVersionFiles(ctx, db, files); err != nil {
		return err
	}
	return checkStickerFiles(ctx, db, files)
}

// sourceIndex maps a sources/<id>/original.<ext> backup path to its id. Every
// source folder holds exactly one original file.
func sourceIndex(files map[string]manifestFile) (map[string]manifestFile, error) {
	bySource := make(map[string]manifestFile)
	for rel, entry := range files {
		parts := strings.Split(rel, "/")
		if len(parts) != 3 || parts[0] != "sources" {
			continue
		}
		if _, duplicate := bySource[parts[1]]; duplicate {
			return nil, fmt.Errorf("%w: source %s has more than one file", ErrBackupInvalid, parts[1])
		}
		bySource[parts[1]] = entry
	}
	return bySource, nil
}

func checkPackSources(ctx context.Context, db *sql.DB, sources map[string]manifestFile) error {
	rows, err := db.QueryContext(ctx, `SELECT id, source_id, source_checksum_sha256, source_byte_size FROM packs WHERE source_id IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("%w: read pack sources: %v", ErrBackupInvalid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			packID, sourceID, checksum string
			byteSize                   int64
		)
		if err := rows.Scan(&packID, &sourceID, &checksum, &byteSize); err != nil {
			return fmt.Errorf("%w: read pack sources: %v", ErrBackupInvalid, err)
		}
		entry, ok := sources[sourceID]
		if !ok {
			return fmt.Errorf("%w: pack %s references missing source %s", ErrBackupInvalid, packID, sourceID)
		}
		if checksum != "" && entry.SHA256 != checksum {
			return fmt.Errorf("%w: source %s does not match its recorded checksum", ErrBackupInvalid, sourceID)
		}
		if byteSize > 0 && entry.ByteSize != byteSize {
			return fmt.Errorf("%w: source %s does not match its recorded size", ErrBackupInvalid, sourceID)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: read pack sources: %v", ErrBackupInvalid, err)
	}
	return nil
}

// checkReferenceAssets validates the private reference images against
// sources/<asset-id>/original.<ext> (spec §8.1). Archived references are
// validated too: the file stays on disk for as long as the row exists.
func checkReferenceAssets(ctx context.Context, db *sql.DB, sources map[string]manifestFile) error {
	rows, err := db.QueryContext(ctx, `SELECT id, byte_size, checksum_sha256 FROM reference_assets`)
	if err != nil {
		return fmt.Errorf("%w: read reference assets: %v", ErrBackupInvalid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, checksum string
			byteSize     int64
		)
		if err := rows.Scan(&id, &byteSize, &checksum); err != nil {
			return fmt.Errorf("%w: read reference assets: %v", ErrBackupInvalid, err)
		}
		entry, ok := sources[id]
		if !ok {
			return fmt.Errorf("%w: reference asset %s has no source file", ErrBackupInvalid, id)
		}
		if entry.SHA256 != checksum {
			return fmt.Errorf("%w: reference asset %s does not match its recorded checksum", ErrBackupInvalid, id)
		}
		if entry.ByteSize != byteSize {
			return fmt.Errorf("%w: reference asset %s does not match its recorded size", ErrBackupInvalid, id)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: read reference assets: %v", ErrBackupInvalid, err)
	}
	return nil
}

func checkVersionFiles(ctx context.Context, db *sql.DB, files map[string]manifestFile) error {
	rows, err := db.QueryContext(ctx, `SELECT pack_id, version, bundle_path, bundle_byte_size, thumbnail_path FROM pack_versions`)
	if err != nil {
		return fmt.Errorf("%w: read pack versions: %v", ErrBackupInvalid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			packID                    string
			version                   int
			bundlePath, thumbnailPath string
			bundleSize                int64
		)
		if err := rows.Scan(&packID, &version, &bundlePath, &bundleSize, &thumbnailPath); err != nil {
			return fmt.Errorf("%w: read pack versions: %v", ErrBackupInvalid, err)
		}
		bundle, err := referencedFile(files, bundlePath)
		if err != nil {
			return err
		}
		if bundle.ByteSize != bundleSize {
			return fmt.Errorf("%w: bundle %s does not match its recorded size", ErrBackupInvalid, bundlePath)
		}
		if _, err := referencedFile(files, thumbnailPath); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: read pack versions: %v", ErrBackupInvalid, err)
	}
	return nil
}

func checkStickerFiles(ctx context.Context, db *sql.DB, files map[string]manifestFile) error {
	rows, err := db.QueryContext(ctx, `SELECT pack_id, version, sort_order, object_path, thumb_path, checksum_sha256 FROM stickers`)
	if err != nil {
		return fmt.Errorf("%w: read stickers: %v", ErrBackupInvalid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			packID                                string
			version, sortOrder                    int
			objectPath, thumbPath, checksumSHA256 string
		)
		if err := rows.Scan(&packID, &version, &sortOrder, &objectPath, &thumbPath, &checksumSHA256); err != nil {
			return fmt.Errorf("%w: read stickers: %v", ErrBackupInvalid, err)
		}
		object, err := referencedFile(files, objectPath)
		if err != nil {
			return err
		}
		if object.SHA256 != checksumSHA256 {
			return fmt.Errorf("%w: sticker %s does not match its recorded checksum", ErrBackupInvalid, objectPath)
		}
		if _, err := referencedFile(files, thumbPath); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: read stickers: %v", ErrBackupInvalid, err)
	}
	return nil
}

// referencedFile resolves a database-held path inside the verified backup.
func referencedFile(files map[string]manifestFile, p string) (manifestFile, error) {
	rel, err := safeBackupPath(p)
	if err != nil {
		return manifestFile{}, err
	}
	entry, ok := files[rel]
	if !ok {
		return manifestFile{}, fmt.Errorf("%w: referenced file %s is missing", ErrBackupInvalid, p)
	}
	return entry, nil
}

// resetCredentials sets the new owner password, deletes every admin session and
// disables every restored API key before the data dir is published.
func resetCredentials(ctx context.Context, db *sql.DB, password []byte) error {
	service := auth.NewService(db)
	err := service.SetPassword(ctx, password)
	if errors.Is(err, auth.ErrNoOwner) {
		err = service.BootstrapOwner(ctx, password)
		if errors.Is(err, auth.ErrOwnerExists) {
			err = nil
		}
	}
	if err != nil {
		return fmt.Errorf("maintenance: set owner password: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM admin_sessions`); err != nil {
		return fmt.Errorf("maintenance: clear admin sessions: %w", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE api_keys SET active = 0, disabled_at = COALESCE(disabled_at, ?)`, utcNow()); err != nil {
		return fmt.Errorf("maintenance: disable api keys: %w", err)
	}
	return nil
}

// ensureEmptyTarget accepts only an empty data dir, apart from the app lock this
// command has just created.
func ensureEmptyTarget(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("maintenance: inspect data dir: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != storage.LockFileName {
			return fmt.Errorf("%w: %s contains %s", ErrDataDirNotEmpty, dir, entry.Name())
		}
	}
	return nil
}

// publish renames the validated staged entries into the data dir without
// overwriting anything. The database goes last: its presence marks a complete
// restore, and the lock file stays where it is.
func publish(stagingDir, dataDir string) error {
	for _, name := range []string{"sources", "packs", "bundles", storage.DBFileName} {
		src := filepath.Join(stagingDir, name)
		if _, err := os.Lstat(src); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("maintenance: inspect %s: %w", src, err)
		}
		dst := filepath.Join(dataDir, name)
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("%w: %s already exists", ErrDataDirNotEmpty, dst)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("maintenance: inspect %s: %w", dst, err)
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("maintenance: publish %s: %w", name, err)
		}
	}
	return syncDir(dataDir)
}
