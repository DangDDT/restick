package maintenance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/storage"
)

const (
	oldPassword     = "old-owner-password"
	newPassword     = "new-owner-password"
	fixturePackID   = "11111111-1111-4111-8111-111111111111"
	fixtureJobID    = "job-maintenance"
	stickerWebP     = "UklGRi4AAABXRUJQVlA4TCIAAAAv/8F/AAdQlHIXpf8BgUCyv/cMRfQ/4z//+c9//vOf//wf"
	thumbWebP       = "UklGRhgAAABXRUJQVlA4TAsAAAAvf8AfAEVRShv9DwA="
	escapingPath    = "../escaped.txt"
	replacementHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// fixture is one committed factory data dir plus the values a restore must
// reproduce, captured before the app is stopped.
type fixture struct {
	dataDir                   string
	oldSession                string
	rotatedKey                string // created, then rotated away: inactive before the backup
	activeKey                 string
	sourceRel                 string
	sourceChecksum            string
	referenceRel              string
	referenceChecksum         string
	archivedReferenceRel      string
	archivedReferenceChecksum string
	objectPath                string
	thumbPath                 string
	thumbnailPath             string
	bundlePath                string
	objectChecksum            string
	thumbChecksum             string
	bundleBytes               []byte
}

// buildFixture produces a real data dir: SQLite rows from storage.Open, a real
// source image from assets.Store, and a committed pack version and bundle built
// by assets.StageBundle/CommitFiles.
func buildFixture(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()

	db, err := storage.Open(ctx, dir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	store, err := assets.NewStore(dir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	source, err := store.SaveSource(ctx, bytes.NewReader(encodePNG(t)), "sheet.png")
	if err != nil {
		t.Fatalf("SaveSource: %v", err)
	}
	reference, err := store.SaveSource(ctx, bytes.NewReader(append(encodePNG(t), 0x01)), "reference.png")
	if err != nil {
		t.Fatalf("SaveSource reference: %v", err)
	}
	archivedReference, err := store.SaveSource(ctx, bytes.NewReader(append(encodePNG(t), 0x02)), "archived-reference.png")
	if err != nil {
		t.Fatalf("SaveSource archived reference: %v", err)
	}
	for id, asset := range map[string]assets.Source{"char-active": reference, "char-archived": archivedReference} {
		execSQL(t, db, `INSERT INTO reference_assets (id, filename, content_type, byte_size, width, height, checksum_sha256)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			asset.ID, asset.Filename, asset.MediaType, asset.ByteSize, asset.Width, asset.Height, asset.ChecksumSHA256)
		execSQL(t, db, `INSERT INTO characters (id, name, reference_asset_id) VALUES (?, ?, ?)`, id, id, asset.ID)
	}
	// Archived characters keep their source file; restore must still validate it.
	execSQL(t, db, `UPDATE characters SET archived_at = '2026-11-10T00:00:00Z' WHERE id = 'char-archived'`)

	service := auth.NewService(db)
	if err := service.BootstrapOwner(ctx, []byte(oldPassword)); err != nil {
		t.Fatalf("BootstrapOwner: %v", err)
	}
	session, err := service.Login(ctx, oldPassword, time.Now())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	first, err := service.CreateKey(ctx, "laptop", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	active, err := service.RotateKey(ctx, first.ID)
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}

	execSQL(t, db, `INSERT INTO packs (id, slug, name, author, status, current_version,
		source_id, source_filename, source_checksum_sha256, source_width, source_height, source_byte_size)
		VALUES (?, 'maintenance-pack', 'Maintenance Pack', 'Tester', 'ready', 1, ?, ?, ?, ?, ?, ?)`,
		fixturePackID, source.ID, source.Filename, source.ChecksumSHA256, source.Width, source.Height, source.ByteSize)

	output := filepath.Join(mustWorkDir(t, store), "output")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatalf("create work output: %v", err)
	}
	writeFile(t, filepath.Join(output, "000.webp"), mustWebP(t, stickerWebP))
	writeFile(t, filepath.Join(output, "thumb_000.webp"), mustWebP(t, thumbWebP))

	manifest := assets.PackManifest{
		PackID:      fixturePackID,
		Slug:        "maintenance-pack",
		Name:        "Maintenance Pack",
		Author:      "Tester",
		Description: "snapshot fixture",
		Tags:        []string{"demo"},
		Version:     1,
		Stickers: []assets.ManifestSticker{{
			File:      "000.webp",
			ThumbFile: "thumb_000.webp",
			Width:     assets.StickerDimension,
			Height:    assets.StickerDimension,
			EmojiHint: "🙂",
			Keywords:  []string{"demo"},
		}},
	}
	staged, err := store.StageBundle(ctx, fixtureJobID, manifest)
	if err != nil {
		t.Fatalf("StageBundle: %v", err)
	}
	committed, err := store.CommitFiles(ctx, staged)
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}

	execSQL(t, db, `INSERT INTO processing_jobs (id, pack_id, version, status, request_json)
		VALUES (?, ?, 1, 'succeeded', '{"schema_version":1,"rows":1,"columns":1}')`, fixtureJobID, fixturePackID)
	execSQL(t, db, `INSERT INTO pack_versions (pack_id, version, job_id, checksum_sha256, sticker_count,
		manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, 1, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fixturePackID, fixtureJobID, committed.ChecksumSHA256, committed.StickerCount,
		string(committed.ManifestJSON), committed.BundlePath, committed.BundleByteSize,
		committed.ThumbnailPath, committed.OutputDir)
	for _, sticker := range committed.Stickers {
		keywords, err := json.Marshal(sticker.Keywords)
		if err != nil {
			t.Fatalf("encode keywords: %v", err)
		}
		execSQL(t, db, `INSERT INTO stickers (pack_id, version, sort_order, filename, object_path, thumb_path,
			checksum_sha256, width, height, emoji_hint, keywords_json) VALUES (?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fixturePackID, sticker.SortOrder, sticker.Filename, sticker.ObjectPath, sticker.ThumbPath,
			sticker.ChecksumSHA256, sticker.Width, sticker.Height, sticker.EmojiHint, string(keywords))
	}

	// Transient work output and a bootstrap secret: neither may reach a backup.
	writeFile(t, filepath.Join(dir, "bootstrap-password"), []byte("bootstrap secret\n"))

	// The app is stopped while backup/restore run.
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}

	bundleBytes, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(committed.BundlePath)))
	if err != nil {
		t.Fatalf("read fixture bundle: %v", err)
	}
	thumbChecksum, _ := digestOf(t, filepath.Join(dir, filepath.FromSlash(committed.Stickers[0].ThumbPath)))

	return fixture{
		dataDir:                   dir,
		oldSession:                session,
		rotatedKey:                first.Raw,
		activeKey:                 active.Raw,
		sourceRel:                 source.RelativePath,
		sourceChecksum:            source.ChecksumSHA256,
		referenceRel:              reference.RelativePath,
		referenceChecksum:         reference.ChecksumSHA256,
		archivedReferenceRel:      archivedReference.RelativePath,
		archivedReferenceChecksum: archivedReference.ChecksumSHA256,
		objectPath:                committed.Stickers[0].ObjectPath,
		thumbPath:                 committed.Stickers[0].ThumbPath,
		thumbnailPath:             committed.ThumbnailPath,
		bundlePath:                committed.BundlePath,
		objectChecksum:            committed.Stickers[0].ChecksumSHA256,
		thumbChecksum:             thumbChecksum,
		bundleBytes:               bundleBytes,
	}
}

func TestBackupRejectsRunningWriter(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := storage.Open(ctx, dir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	release, err := storage.AcquireAppLock(dir)
	if err != nil {
		t.Fatalf("AcquireAppLock: %v", err)
	}
	defer release()

	out := filepath.Join(t.TempDir(), "backup")
	if err := Backup(ctx, dir, out); !errors.Is(err, storage.ErrAppLocked) {
		t.Fatalf("Backup error = %v, want ErrAppLocked", err)
	}
	if _, err := os.Lstat(out); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("backup output exists after a rejected backup: %v", err)
	}
}

func TestRestoreRejectsRunningWriter(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)
	out := backupFixture(t, f)
	target := t.TempDir()
	release, err := storage.AcquireAppLock(target)
	if err != nil {
		t.Fatalf("AcquireAppLock: %v", err)
	}
	defer release()

	if err := Restore(ctx, out, target, writePassword(t, newPassword)); !errors.Is(err, storage.ErrAppLocked) {
		t.Fatalf("Restore error = %v, want ErrAppLocked", err)
	}
	assertOnlyLock(t, target)
}

func TestBackupManifestCoversSnapshot(t *testing.T) {
	f := buildFixture(t)
	out := backupFixture(t, f)

	manifest := loadManifest(t, out)
	if manifest.Schema != BackupSchema {
		t.Fatalf("schema = %q, want %q", manifest.Schema, BackupSchema)
	}
	listed := make(map[string]manifestFile, len(manifest.Files))
	for _, entry := range manifest.Files {
		listed[entry.Path] = entry
	}
	for _, want := range []string{storage.DBFileName, f.sourceRel, f.objectPath, f.thumbPath, f.bundlePath, f.thumbnailPath} {
		if _, ok := listed[want]; !ok {
			t.Errorf("manifest does not list %s", want)
		}
	}
	for rel, entry := range listed {
		if strings.HasPrefix(rel, "work/") || !strings.HasPrefix(rel, "sources/") &&
			!strings.HasPrefix(rel, "packs/") && !strings.HasPrefix(rel, "bundles/") && rel != storage.DBFileName {
			t.Errorf("manifest lists %s, which is outside the backup allowlist", rel)
		}
		digest, size := digestOf(t, filepath.Join(out, filepath.FromSlash(rel)))
		if digest != entry.SHA256 || size != entry.ByteSize {
			t.Errorf("%s: manifest digest does not match the backup file", rel)
		}
	}
	if _, err := os.Lstat(filepath.Join(out, "work")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("backup contains work/: %v", err)
	}
}

func TestBackupOutputMustBeEmptyOutsideDataDir(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)

	t.Run("existing empty dir", func(t *testing.T) {
		out := t.TempDir()
		if err := Backup(ctx, f.dataDir, out); err != nil {
			t.Fatalf("Backup: %v", err)
		}
		if manifest := loadManifest(t, out); manifest.Schema != BackupSchema {
			t.Fatalf("schema = %q, want %q", manifest.Schema, BackupSchema)
		}
	})

	t.Run("non-empty dir", func(t *testing.T) {
		out := t.TempDir()
		writeFile(t, filepath.Join(out, "stale"), []byte("x"))
		if err := Backup(ctx, f.dataDir, out); !errors.Is(err, ErrInvalidBackupOutput) {
			t.Fatalf("Backup error = %v, want ErrInvalidBackupOutput", err)
		}
	})

	t.Run("inside data dir", func(t *testing.T) {
		if err := Backup(ctx, f.dataDir, filepath.Join(f.dataDir, "backup")); !errors.Is(err, ErrInvalidBackupOutput) {
			t.Fatalf("Backup error = %v, want ErrInvalidBackupOutput", err)
		}
	})

	t.Run("symlink into data dir", func(t *testing.T) {
		inside := filepath.Join(f.dataDir, "exports")
		if err := os.MkdirAll(inside, 0o755); err != nil {
			t.Fatalf("create data dir subdir: %v", err)
		}
		link := filepath.Join(t.TempDir(), "backup-link")
		if err := os.Symlink(inside, link); err != nil {
			t.Fatalf("symlink output: %v", err)
		}
		if err := Backup(ctx, f.dataDir, link); !errors.Is(err, ErrInvalidBackupOutput) {
			t.Fatalf("Backup error = %v, want ErrInvalidBackupOutput", err)
		}
		entries, err := os.ReadDir(inside)
		if err != nil {
			t.Fatalf("read data dir subdir: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("backup wrote %d entries inside the data dir", len(entries))
		}
	})
}

func TestRestoreRefusesExistingData(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)
	out := backupFixture(t, f)
	password := writePassword(t, newPassword)

	for _, name := range []string{"database", "resource dir"} {
		t.Run(name, func(t *testing.T) {
			target := t.TempDir()
			var existing string
			if name == "database" {
				existing = filepath.Join(target, storage.DBFileName)
				writeFile(t, existing, []byte("existing database"))
			} else {
				existing = filepath.Join(target, "sources")
				if err := os.MkdirAll(existing, 0o755); err != nil {
					t.Fatalf("create existing dir: %v", err)
				}
			}
			if err := Restore(ctx, out, target, password); !errors.Is(err, ErrDataDirNotEmpty) {
				t.Fatalf("Restore error = %v, want ErrDataDirNotEmpty", err)
			}
			if name == "database" {
				body, err := os.ReadFile(existing)
				if err != nil || string(body) != "existing database" {
					t.Fatalf("existing database was modified: %q, %v", body, err)
				}
			}
		})
	}

	t.Run("restored data dir", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "data")
		if err := Restore(ctx, out, target, password); err != nil {
			t.Fatalf("first Restore: %v", err)
		}
		if err := Restore(ctx, out, target, password); !errors.Is(err, ErrDataDirNotEmpty) {
			t.Fatalf("second Restore error = %v, want ErrDataDirNotEmpty", err)
		}
	})
}

func TestRestoreRejectsTamperedOrEscapingFiles(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)
	base := backupFixture(t, f)
	password := writePassword(t, newPassword)

	cases := []struct {
		name   string
		want   string
		tamper func(t *testing.T, backupDir string)
	}{
		{
			name: "tampered file bytes",
			tamper: func(t *testing.T, backupDir string) {
				target := filepath.Join(backupDir, filepath.FromSlash(f.objectPath))
				body, err := os.ReadFile(target)
				if err != nil {
					t.Fatalf("read sticker: %v", err)
				}
				writeFile(t, target, append(body, 0x00))
			},
		},
		{
			name: "escaping manifest path",
			tamper: func(t *testing.T, backupDir string) {
				editManifest(t, backupDir, func(m *backupManifest) {
					setEntry(m, escapingPath, replacementHash, 1)
				})
			},
		},
		{
			name: "symlink leaving the backup",
			tamper: func(t *testing.T, backupDir string) {
				outside := filepath.Join(t.TempDir(), "outside.webp")
				writeFile(t, outside, []byte("not a sticker"))
				object := filepath.Join(backupDir, filepath.FromSlash(f.objectPath))
				if err := os.Remove(object); err != nil {
					t.Fatalf("remove sticker: %v", err)
				}
				if err := os.Symlink(outside, object); err != nil {
					t.Fatalf("symlink sticker: %v", err)
				}
				digest, size := digestOf(t, outside)
				editManifest(t, backupDir, func(m *backupManifest) {
					setEntry(m, f.objectPath, digest, size)
				})
			},
		},
		{
			name: "database referencing a missing file",
			tamper: func(t *testing.T, backupDir string) {
				if err := os.Remove(filepath.Join(backupDir, filepath.FromSlash(f.objectPath))); err != nil {
					t.Fatalf("remove sticker: %v", err)
				}
				editManifest(t, backupDir, func(m *backupManifest) {
					removeEntry(m, f.objectPath)
				})
			},
		},
		{
			name: "corrupted database bytes",
			tamper: func(t *testing.T, backupDir string) {
				dbPath := filepath.Join(backupDir, storage.DBFileName)
				body, err := os.ReadFile(dbPath)
				if err != nil {
					t.Fatalf("read database: %v", err)
				}
				writeFile(t, dbPath, body[:100])
				digest, size := digestOf(t, dbPath)
				editManifest(t, backupDir, func(m *backupManifest) {
					setEntry(m, storage.DBFileName, digest, size)
				})
			},
		},
		{
			name: "missing archived reference source",
			want: "has no source file",
			tamper: func(t *testing.T, backupDir string) {
				if err := os.Remove(filepath.Join(backupDir, filepath.FromSlash(f.archivedReferenceRel))); err != nil {
					t.Fatalf("remove reference asset: %v", err)
				}
				editManifest(t, backupDir, func(m *backupManifest) {
					removeEntry(m, f.archivedReferenceRel)
				})
			},
		},
		{
			name: "replaced reference asset bytes",
			want: "does not match its recorded checksum",
			tamper: func(t *testing.T, backupDir string) {
				reference := filepath.Join(backupDir, filepath.FromSlash(f.referenceRel))
				body, err := os.ReadFile(reference)
				if err != nil {
					t.Fatalf("read reference asset: %v", err)
				}
				writeFile(t, reference, append(body, 0x00))
				digest, size := digestOf(t, reference)
				editManifest(t, backupDir, func(m *backupManifest) {
					setEntry(m, f.referenceRel, digest, size)
				})
			},
		},
		{
			name: "unknown manifest schema",
			tamper: func(t *testing.T, backupDir string) {
				editManifest(t, backupDir, func(m *backupManifest) { m.Schema = "restick.backup.v2" })
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backupDir := filepath.Join(t.TempDir(), "backup")
			copyTree(t, base, backupDir)
			tc.tamper(t, backupDir)

			target := filepath.Join(t.TempDir(), "data")
			err := Restore(ctx, backupDir, target, password)
			if !errors.Is(err, ErrBackupInvalid) {
				t.Fatalf("Restore error = %v, want ErrBackupInvalid", err)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Restore error = %v, want it to mention %q", err, tc.want)
			}
			assertOnlyLock(t, target)
		})
	}
}

func TestRestoreRejectsIncompatibleBackups(t *testing.T) {
	f := buildFixture(t)
	base := backupFixture(t, f)
	password := writePassword(t, newPassword)
	for _, name := range []string{"renamed source", "unsupported source extension", "newer schema"} {
		t.Run(name, func(t *testing.T) {
			backupDir := filepath.Join(t.TempDir(), "backup")
			copyTree(t, base, backupDir)
			if name == "newer schema" {
				dbPath := filepath.Join(backupDir, storage.DBFileName)
				db, err := sql.Open("sqlite", dbPath)
				if err != nil {
					t.Fatal(err)
				}
				execSQL(t, db, `INSERT INTO schema_migrations (version, name) VALUES (2, 'future_schema')`)
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
				digest, size := digestOf(t, dbPath)
				editManifest(t, backupDir, func(m *backupManifest) {
					setEntry(m, storage.DBFileName, digest, size)
				})
			} else {
				basename := "renamed.png"
				if name == "unsupported source extension" {
					basename = "original.gif"
				}
				rel := filepath.ToSlash(filepath.Join(filepath.Dir(f.sourceRel), basename))
				if err := os.Rename(filepath.Join(backupDir, filepath.FromSlash(f.sourceRel)),
					filepath.Join(backupDir, filepath.FromSlash(rel))); err != nil {
					t.Fatal(err)
				}
				editManifest(t, backupDir, func(m *backupManifest) {
					for i := range m.Files {
						if m.Files[i].Path == f.sourceRel {
							m.Files[i].Path = rel
						}
					}
				})
			}
			target := filepath.Join(t.TempDir(), "data")
			if err := Restore(context.Background(), backupDir, target, password); !errors.Is(err, ErrBackupInvalid) {
				t.Fatalf("Restore error = %v, want ErrBackupInvalid", err)
			}
			assertOnlyLock(t, target)
		})
	}
}

func TestRestoredFilesMatchDatabase(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)
	out := backupFixture(t, f)
	target := filepath.Join(t.TempDir(), "data")
	if err := Restore(ctx, out, target, writePassword(t, newPassword)); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	// The published data dir is a checkpointed database, not a WAL pair.
	if _, err := os.Lstat(filepath.Join(target, storage.DBFileName+"-wal")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("restored data dir kept a WAL file: %v", err)
	}

	db, err := storage.Open(ctx, target)
	if err != nil {
		t.Fatalf("open restored database: %v", err)
	}
	defer db.Close()

	var status string
	var currentVersion int
	if err := db.QueryRow(`SELECT status, current_version FROM packs WHERE id = ?`, fixturePackID).
		Scan(&status, &currentVersion); err != nil {
		t.Fatalf("read pack: %v", err)
	}
	if status != "ready" || currentVersion != 1 {
		t.Fatalf("pack = %s/v%d, want ready/v1", status, currentVersion)
	}

	var bundlePath string
	var bundleSize int64
	if err := db.QueryRow(`SELECT bundle_path, bundle_byte_size FROM pack_versions WHERE pack_id = ? AND version = 1`, fixturePackID).
		Scan(&bundlePath, &bundleSize); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if bundlePath != f.bundlePath || bundleSize != int64(len(f.bundleBytes)) {
		t.Fatalf("version = %s/%d, want %s/%d", bundlePath, bundleSize, f.bundlePath, len(f.bundleBytes))
	}

	var objectPath, thumbPath, checksum string
	var stickerCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stickers`).Scan(&stickerCount); err != nil {
		t.Fatalf("count stickers: %v", err)
	}
	if stickerCount != 1 {
		t.Fatalf("sticker count = %d, want 1", stickerCount)
	}
	if err := db.QueryRow(`SELECT object_path, thumb_path, checksum_sha256 FROM stickers`).
		Scan(&objectPath, &thumbPath, &checksum); err != nil {
		t.Fatalf("read sticker: %v", err)
	}
	if objectPath != f.objectPath || thumbPath != f.thumbPath || checksum != f.objectChecksum {
		t.Fatalf("sticker row = %s/%s/%s, want %s/%s/%s", objectPath, thumbPath, checksum, f.objectPath, f.thumbPath, f.objectChecksum)
	}

	var sourceID, sourceChecksum string
	if err := db.QueryRow(`SELECT source_id, source_checksum_sha256 FROM packs WHERE id = ?`, fixturePackID).
		Scan(&sourceID, &sourceChecksum); err != nil {
		t.Fatalf("read pack source: %v", err)
	}
	if sourceChecksum != f.sourceChecksum {
		t.Fatalf("source checksum = %s, want %s", sourceChecksum, f.sourceChecksum)
	}

	restoredBundle, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(f.bundlePath)))
	if err != nil {
		t.Fatalf("read restored bundle: %v", err)
	}
	if !bytes.Equal(restoredBundle, f.bundleBytes) {
		t.Fatalf("restored bundle bytes differ from the committed fixture bundle")
	}
	for _, check := range []struct{ rel, digest string }{
		{f.objectPath, f.objectChecksum},
		{f.thumbPath, f.thumbChecksum},
		{f.thumbnailPath, f.thumbChecksum},
		{f.sourceRel, f.sourceChecksum},
		{f.referenceRel, f.referenceChecksum},
		{f.archivedReferenceRel, f.archivedReferenceChecksum},
	} {
		digest, _ := digestOf(t, filepath.Join(target, filepath.FromSlash(check.rel)))
		if digest != check.digest {
			t.Errorf("%s digest = %s, want %s", check.rel, digest, check.digest)
		}
	}

	if _, err := os.Lstat(filepath.Join(target, "work")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("restore copied work/: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(target, "bootstrap-password")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("restore copied the bootstrap secret: %v", err)
	}
}

func TestRestoreInvalidatesCredentials(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)
	out := backupFixture(t, f)
	target := filepath.Join(t.TempDir(), "data")
	if err := Restore(ctx, out, target, writePassword(t, newPassword)); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	db, err := storage.Open(ctx, target)
	if err != nil {
		t.Fatalf("open restored database: %v", err)
	}
	defer db.Close()
	service := auth.NewService(db)
	now := time.Now()

	if err := service.ValidateSession(ctx, f.oldSession, now); !errors.Is(err, auth.ErrSessionInvalid) {
		t.Fatalf("old session error = %v, want ErrSessionInvalid", err)
	}
	for name, raw := range map[string]string{"rotated key": f.rotatedKey, "active key": f.activeKey} {
		if err := service.ValidateKey(ctx, raw, now); !errors.Is(err, auth.ErrKeyNotFound) {
			t.Fatalf("%s error = %v, want ErrKeyNotFound", name, err)
		}
	}

	var activeKeys, sessions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE active = 1`).Scan(&activeKeys); err != nil {
		t.Fatalf("count active keys: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_sessions`).Scan(&sessions); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if activeKeys != 0 || sessions != 0 {
		t.Fatalf("active keys = %d, sessions = %d, want 0 and 0", activeKeys, sessions)
	}

	if _, err := service.Login(ctx, oldPassword, now); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("old password error = %v, want ErrInvalidCredentials", err)
	}
	session, err := service.Login(ctx, newPassword, now)
	if err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	if err := service.ValidateSession(ctx, session, now); err != nil {
		t.Fatalf("new session invalid: %v", err)
	}
}

func TestRestoreMissingPasswordFailsBeforeChanges(t *testing.T) {
	ctx := context.Background()
	f := buildFixture(t)
	out := backupFixture(t, f)

	missing := filepath.Join(t.TempDir(), "no-such-password")
	empty := filepath.Join(t.TempDir(), "empty-password")
	writeFile(t, empty, nil)
	blank := filepath.Join(t.TempDir(), "blank-password")
	writeFile(t, blank, []byte("  \n\n"))

	for name, passwordFile := range map[string]string{
		"missing file":    missing,
		"empty file":      empty,
		"whitespace file": blank,
	} {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "data")
			if err := Restore(ctx, out, target, passwordFile); !errors.Is(err, ErrMissingPassword) {
				t.Fatalf("Restore error = %v, want ErrMissingPassword", err)
			}
			if _, err := os.Lstat(target); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("data dir was created before the password check: %v", err)
			}
		})
	}
}

func backupFixture(t *testing.T, f fixture) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "backup")
	if err := Backup(context.Background(), f.dataDir, out); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	return out
}

func writePassword(t *testing.T, password string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "new-password")
	writeFile(t, p, []byte(password+"\n"))
	return p
}

func loadManifest(t *testing.T, backupDir string) backupManifest {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(backupDir, manifestFileName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}

func editManifest(t *testing.T, backupDir string, edit func(*backupManifest)) {
	t.Helper()
	manifest := loadManifest(t, backupDir)
	edit(&manifest)
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("encode manifest: %v", err)
	}
	writeFile(t, filepath.Join(backupDir, manifestFileName), append(body, '\n'))
}

func setEntry(m *backupManifest, rel, digest string, size int64) {
	for i := range m.Files {
		if m.Files[i].Path == rel {
			m.Files[i].SHA256 = digest
			m.Files[i].ByteSize = size
			return
		}
	}
	m.Files = append(m.Files, manifestFile{Path: rel, ByteSize: size, SHA256: digest})
}

func removeEntry(m *backupManifest, rel string) {
	kept := m.Files[:0]
	for _, entry := range m.Files {
		if entry.Path != rel {
			kept = append(kept, entry)
		}
	}
	m.Files = kept
}

// copyTree duplicates a backup folder so one subtest cannot disturb another.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatalf("copy backup: %v", err)
	}
}

func assertOnlyLock(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatalf("read data dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != storage.LockFileName {
			t.Fatalf("failed restore left %s in the data dir", entry.Name())
		}
	}
}

func digestOf(t *testing.T, p string) (string, int64) {
	t.Helper()
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), int64(len(body))
}

func writeFile(t *testing.T, p string, body []byte) {
	t.Helper()
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func mustWorkDir(t *testing.T, store *assets.Store) string {
	t.Helper()
	dir, err := store.WorkDir(fixtureJobID)
	if err != nil {
		t.Fatalf("WorkDir: %v", err)
	}
	return dir
}

func mustWebP(t *testing.T, encoded string) []byte {
	t.Helper()
	body, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode webp fixture: %v", err)
	}
	return body
}

func encodePNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
