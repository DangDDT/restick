// Package maintenance implements the offline factory backup and restore behind
// `restick backup` and `restick restore` (spec §12).
//
// Threat model of restick.backup.v1: the manifest records a SHA256 and byte size
// for every file, so restore detects corrupted or edited bytes. It does NOT
// authenticate the backup — whoever can write the backup folder can rewrite the
// manifest and its digests — so only restore a backup produced on this instance
// and keep it protected outside the product.
package maintenance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/eykh/restick/api_svc/internal/storage"
	_ "modernc.org/sqlite"
)

const (
	// BackupSchema is the backup folder format written by Backup; Restore
	// refuses every other schema.
	BackupSchema = "restick.backup.v1"

	// manifestFileName lists the backed-up files inside the backup folder.
	manifestFileName = "manifest.json"

	// dbFileName and lockFileName mirror internal/storage. Backup snapshots the
	// database by name, and restore must tolerate the app lock it creates.
	dbFileName   = "restick.sqlite"
	lockFileName = ".restick.lock"
)

// resourceDirs are the only data-dir subtrees a backup contains. work/ holds
// transient job artifacts and is never backed up.
var resourceDirs = []string{"sources", "packs", "bundles"}

var (
	// ErrBackupInvalid reports a backup folder or restored database that fails
	// manifest, path, checksum, schema, integrity, foreign key or referenced-file
	// validation.
	ErrBackupInvalid = errors.New("maintenance: invalid backup")
	// ErrDataDirNotEmpty reports a restore target that already holds app data.
	ErrDataDirNotEmpty = errors.New("maintenance: data dir is not empty")
	// ErrMissingPassword reports an absent or blank new-owner-password file.
	ErrMissingPassword = errors.New("maintenance: new password file is missing or empty")
	// ErrInvalidBackupOutput reports an unusable backup output directory.
	ErrInvalidBackupOutput = errors.New("maintenance: invalid backup output directory")
	// ErrNoDatabase reports a data dir that holds no factory database.
	ErrNoDatabase = errors.New("maintenance: data dir has no restick.sqlite")
)

// backupManifest is the complete description of one backup folder.
type backupManifest struct {
	Schema    string         `json:"schema"`
	CreatedAt string         `json:"created_at"`
	Files     []manifestFile `json:"files"`
}

// manifestFile is one backed-up file, addressed relative to the backup root.
type manifestFile struct {
	Path     string `json:"path"`
	ByteSize int64  `json:"byte_size"`
	SHA256   string `json:"sha256"`
}

// Backup writes an offline snapshot of dataDir into a new outputDir:
// restick.sqlite, sources/, packs/ and bundles/ plus manifest.json. work/ and
// every other entry of the data dir are left out.
//
// dataDir must not be in use by a serving app: Backup takes the same exclusive
// app lock as the server and fails with storage.ErrAppLocked when a writer
// already holds it. The database is checkpointed and the data dir must already
// contain restick.sqlite; there is no empty or in-memory fallback.
//
// outputDir must be absent or empty and outside dataDir, because the data dir is
// the static resource root. Files are staged inside outputDir and manifest.json
// is published last, so a folder without a manifest is not a backup and never
// passes validation.
func Backup(ctx context.Context, dataDir, outputDir string) error {
	dir, err := canonicalDir(dataDir)
	if err != nil {
		return err
	}
	out, err := absolutePath(outputDir)
	if err != nil {
		return err
	}
	if within(dir, out) || within(dir, resolvedTarget(out)) {
		return fmt.Errorf("%w: %s is inside the data dir", ErrInvalidBackupOutput, out)
	}
	switch entries, err := os.ReadDir(out); {
	case err == nil && len(entries) > 0:
		return fmt.Errorf("%w: %s is not empty", ErrInvalidBackupOutput, out)
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %v", ErrInvalidBackupOutput, err)
	}

	release, err := storage.AcquireAppLock(dir)
	if err != nil {
		return err
	}
	defer release()

	if err := ctx.Err(); err != nil {
		return err
	}
	dbPath := filepath.Join(dir, dbFileName)
	if _, err := os.Lstat(dbPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNoDatabase, dbPath)
		}
		return fmt.Errorf("maintenance: inspect %s: %w", dbPath, err)
	}
	if err := checkpoint(ctx, dbPath); err != nil {
		return err
	}

	// Staging lives inside outputDir so a mounted backup volume works too.
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("maintenance: prepare backup output: %w", err)
	}
	staging, err := os.MkdirTemp(out, ".restick-backup-")
	if err != nil {
		return fmt.Errorf("maintenance: create backup staging: %w", err)
	}
	defer os.RemoveAll(staging)

	manifest := backupManifest{Schema: BackupSchema, CreatedAt: utcNow()}
	if err := snapshotEntry(ctx, dir, staging, dbFileName, &manifest.Files); err != nil {
		return err
	}
	for _, name := range resourceDirs {
		if err := snapshotEntry(ctx, dir, staging, name, &manifest.Files); err != nil {
			return err
		}
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })

	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("maintenance: encode manifest: %w", err)
	}
	if err := writeFileSync(filepath.Join(staging, manifestFileName), append(body, '\n')); err != nil {
		return err
	}
	if err := syncDir(staging); err != nil {
		return err
	}
	if err := publishBackup(staging, out); err != nil {
		return err
	}
	return syncDir(out)
}

// publishBackup moves the staged snapshot into the output dir, manifest last:
// its presence is what makes the folder a complete backup.
func publishBackup(staging, out string) error {
	entries, err := os.ReadDir(staging)
	if err != nil {
		return fmt.Errorf("maintenance: read backup staging: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() != manifestFileName && entries[j].Name() == manifestFileName
	})
	for _, entry := range entries {
		if err := os.Rename(filepath.Join(staging, entry.Name()), filepath.Join(out, entry.Name())); err != nil {
			return fmt.Errorf("maintenance: publish %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// checkpoint flushes the WAL into the database file so the copied file is the
// whole committed state. The app lock proves no other process is writing.
func checkpoint(ctx context.Context, dbPath string) error {
	u := url.URL{Scheme: "file", Path: dbPath}
	db, err := sql.Open("sqlite", u.String()+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("maintenance: open %s: %w", dbPath, err)
	}
	defer db.Close()

	var busy, logFrames, checkpointed int
	if err := db.QueryRowContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &logFrames, &checkpointed); err != nil {
		return fmt.Errorf("maintenance: checkpoint %s: %w", dbPath, err)
	}
	if busy != 0 {
		return fmt.Errorf("maintenance: checkpoint %s: database is busy", dbPath)
	}
	return nil
}

// snapshotEntry copies one top-level data-dir entry into staging and appends its
// files to the manifest. Symlinks and other non-regular entries are rejected
// instead of followed.
func snapshotEntry(ctx context.Context, dataDir, staging, name string, files *[]manifestFile) error {
	src := filepath.Join(dataDir, name)
	info, err := os.Lstat(src)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("maintenance: inspect %s: %w", src, err)
	}
	switch {
	case info.Mode().IsRegular():
		entry, err := copyVerified(ctx, src, filepath.Join(staging, name), name)
		if err != nil {
			return err
		}
		*files = append(*files, entry)
		return nil
	case !info.IsDir():
		return fmt.Errorf("maintenance: %s is not a regular file or directory", src)
	}

	return filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("maintenance: walk %s: %w", p, walkErr)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(dataDir, p)
		if err != nil {
			return fmt.Errorf("maintenance: relative path of %s: %w", p, err)
		}
		rel = filepath.ToSlash(rel)
		dst := filepath.Join(staging, filepath.FromSlash(rel))
		if d.IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("maintenance: create %s: %w", dst, err)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("maintenance: %s is not a regular file", p)
		}
		entry, err := copyVerified(ctx, p, dst, rel)
		if err != nil {
			return err
		}
		*files = append(*files, entry)
		return nil
	})
}

// copyVerified copies one regular file and returns its manifest entry.
func copyVerified(ctx context.Context, src, dst, rel string) (manifestFile, error) {
	sum, size, err := copyFile(ctx, src, dst)
	if err != nil {
		return manifestFile{}, err
	}
	return manifestFile{Path: rel, ByteSize: size, SHA256: sum}, nil
}

// copyFile copies src to a new dst, fsyncs it and returns the SHA256 of the
// copied bytes. src is opened without following a final symlink.
func copyFile(ctx context.Context, src, dst string) (digest string, size int64, err error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	in, err := os.OpenFile(src, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", 0, fmt.Errorf("maintenance: open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", 0, fmt.Errorf("maintenance: create %s: %w", dst, err)
	}
	sum := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, sum), in)
	if copyErr == nil {
		copyErr = out.Sync()
	}
	if closeErr := out.Close(); copyErr == nil && closeErr != nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return "", 0, fmt.Errorf("maintenance: copy %s: %w", src, copyErr)
	}
	return hex.EncodeToString(sum.Sum(nil)), written, nil
}

func writeFileSync(p string, data []byte) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("maintenance: create %s: %w", p, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("maintenance: write %s: %w", p, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("maintenance: fsync %s: %w", p, err)
	}
	return f.Close()
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("maintenance: open dir %s: %w", dir, err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("maintenance: fsync dir %s: %w", dir, err)
	}
	return nil
}

// utcNow is the storage timestamp format: UTC, second precision.
func utcNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func absolutePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errors.New("maintenance: path is required")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("maintenance: resolve %s: %w", p, err)
	}
	return abs, nil
}

// canonicalDir resolves p and every symlink in it; the path must exist.
func canonicalDir(p string) (string, error) {
	abs, err := absolutePath(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("maintenance: resolve %s: %w", p, err)
	}
	return resolved, nil
}

// resolvedTarget resolves the symlinks of p's parent, for a p that does not
// exist yet.
func resolvedTarget(p string) string {
	parent, err := filepath.EvalSymlinks(filepath.Dir(p))
	if err != nil {
		return p
	}
	return filepath.Join(parent, filepath.Base(p))
}

// within reports whether p is root or lies below it.
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
