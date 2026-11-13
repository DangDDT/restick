package packs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/eykh/restick/api_svc/internal/assets"
)

var (
	// ErrJobNotRunning means the job is no longer running (interrupted,
	// revoked or already published), so nothing was committed.
	ErrJobNotRunning = errors.New("packs: job is not running")
	// ErrVersionTaken means the reserved version was already committed.
	ErrVersionTaken = errors.New("packs: version is already committed")
)

const publicationSchema = "restick.job-publication.v1"

// ProcessedVersion is a fully written version whose files were renamed into
// their final public locations but that no database row references yet.
type ProcessedVersion struct {
	JobID       string
	PackID      string
	Version     int
	Manifest    assets.PackManifest
	Files       assets.CommittedFiles
	Diagnostics string
}

// CommitProcessed publishes a processed version in one transaction: immutable
// version and sticker rows, the current pointer, the pack status and the job
// result all commit together. It only commits while the job is still running
// and the pack is not revoked, so a revoke or a terminal state always wins over
// a late completion.
func CommitProcessed(ctx context.Context, db *sql.DB, result ProcessedVersion) error {
	if err := validateProcessed(result); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM packs WHERE id = ?`, result.PackID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("packs: read pack status: %w", err)
	}
	if status == string(StatusRevoked) {
		return ErrRevoked
	}
	var jobStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM processing_jobs WHERE id = ? AND pack_id = ? AND version = ?`,
		result.JobID, result.PackID, result.Version).Scan(&jobStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrJobNotRunning
	}
	if err != nil {
		return fmt.Errorf("packs: read job status: %w", err)
	}
	if jobStatus != "running" {
		return ErrJobNotRunning
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO pack_versions
		(pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json,
		 bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.PackID, result.Version, result.JobID, result.Files.ChecksumSHA256,
		result.Files.StickerCount, string(result.Files.ManifestJSON),
		result.Files.BundlePath, result.Files.BundleByteSize,
		result.Files.ThumbnailPath, result.Files.OutputDir); err != nil {
		if isVersionConflict(err) {
			return ErrVersionTaken
		}
		return fmt.Errorf("packs: insert version: %w", err)
	}
	for _, sticker := range result.Files.Stickers {
		keywords, err := encodeKeywords(sticker.Keywords)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO stickers
			(pack_id, version, sort_order, filename, object_path, thumb_path,
			 checksum_sha256, width, height, emoji_hint, keywords_json,
			 animated_path, animated_checksum_sha256, animated_byte_size,
			 animated_frames, animated_duration_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			result.PackID, result.Version, sticker.SortOrder, sticker.Filename,
			sticker.ObjectPath, sticker.ThumbPath, sticker.ChecksumSHA256,
			sticker.Width, sticker.Height, sticker.EmojiHint, keywords,
			sticker.AnimatedPath, sticker.AnimatedChecksumSHA256, sticker.AnimatedByteSize,
			sticker.AnimatedFrames, sticker.AnimatedDurationMS); err != nil {
			return fmt.Errorf("packs: insert sticker: %w", err)
		}
	}

	updated, err := tx.ExecContext(ctx, `UPDATE packs SET status = ?, current_version = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND status != 'revoked'`, string(StatusReady), result.Version, result.PackID)
	if err != nil {
		return fmt.Errorf("packs: update pack: %w", err)
	}
	if affected, err := updated.RowsAffected(); err != nil {
		return fmt.Errorf("packs: update pack: %w", err)
	} else if affected != 1 {
		return ErrRevoked
	}

	summary, err := json.Marshal(publicationDocument{
		Schema:         publicationSchema,
		JobID:          result.JobID,
		PackID:         result.PackID,
		Version:        result.Version,
		ChecksumSHA256: result.Files.ChecksumSHA256,
		StickerCount:   result.Files.StickerCount,
		BundlePath:     result.Files.BundlePath,
		BundleByteSize: result.Files.BundleByteSize,
		ThumbnailPath:  result.Files.ThumbnailPath,
		OutputDir:      result.Files.OutputDir,
	})
	if err != nil {
		return fmt.Errorf("packs: encode job result: %w", err)
	}
	finished, err := tx.ExecContext(ctx, `UPDATE processing_jobs SET status = 'succeeded',
		result_json = ?, diagnostics = ?,
		finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND status = 'running'`, string(summary), result.Diagnostics, result.JobID)
	if err != nil {
		return fmt.Errorf("packs: mark job succeeded: %w", err)
	}
	if affected, err := finished.RowsAffected(); err != nil {
		return fmt.Errorf("packs: mark job succeeded: %w", err)
	} else if affected != 1 {
		return ErrJobNotRunning
	}
	return tx.Commit()
}

// publicationDocument is the processing_jobs.result_json snapshot of a
// published version.
type publicationDocument struct {
	Schema         string `json:"schema"`
	JobID          string `json:"job_id"`
	PackID         string `json:"pack_id"`
	Version        int    `json:"version"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	StickerCount   int    `json:"sticker_count"`
	BundlePath     string `json:"bundle_path"`
	BundleByteSize int64  `json:"bundle_byte_size"`
	ThumbnailPath  string `json:"thumbnail_path"`
	OutputDir      string `json:"output_dir"`
}

func validateProcessed(result ProcessedVersion) error {
	files := result.Files
	switch {
	case !isUUID(result.JobID) || !isUUID(result.PackID):
		return fmt.Errorf("%w: invalid job or pack id", ErrInvalid)
	case result.Version < 1:
		return fmt.Errorf("%w: invalid version", ErrInvalid)
	case result.Manifest.PackID != result.PackID || result.Manifest.Version != result.Version:
		return fmt.Errorf("%w: manifest does not match the publication", ErrInvalid)
	case files.JobID != result.JobID || files.PackID != result.PackID || files.Version != result.Version:
		return fmt.Errorf("%w: files do not match the publication", ErrInvalid)
	case !checksumPattern.MatchString(files.ChecksumSHA256):
		return fmt.Errorf("%w: invalid version checksum", ErrInvalid)
	case files.StickerCount < 1 || files.StickerCount != len(files.Stickers) || files.StickerCount != len(result.Manifest.Stickers):
		return fmt.Errorf("%w: sticker count does not match", ErrInvalid)
	case len(files.ManifestJSON) == 0 || !json.Valid(files.ManifestJSON):
		return fmt.Errorf("%w: invalid manifest json", ErrInvalid)
	case files.BundlePath == "" || files.ThumbnailPath == "" || files.OutputDir == "" || files.BundleByteSize < 1:
		return fmt.Errorf("%w: missing committed file metadata", ErrInvalid)
	}
	for i, sticker := range files.Stickers {
		if sticker.SortOrder != i {
			return fmt.Errorf("%w: sticker %d is out of order", ErrInvalid, i)
		}
		switch {
		case sticker.Filename == "" || sticker.ObjectPath == "" || sticker.ThumbPath == "":
			return fmt.Errorf("%w: sticker %d has no committed path", ErrInvalid, i)
		case !checksumPattern.MatchString(sticker.ChecksumSHA256):
			return fmt.Errorf("%w: sticker %d has an invalid checksum", ErrInvalid, i)
		case sticker.Width < 1 || sticker.Height < 1:
			return fmt.Errorf("%w: sticker %d has invalid dimensions", ErrInvalid, i)
		case sticker.AnimatedPath != "" && (!checksumPattern.MatchString(sticker.AnimatedChecksumSHA256) ||
			sticker.AnimatedFrames < 2 || sticker.AnimatedByteSize < 1):
			return fmt.Errorf("%w: sticker %d has an invalid animated variant", ErrInvalid, i)
		}
	}
	return nil
}

func encodeKeywords(keywords []string) (string, error) {
	if len(keywords) == 0 {
		return "[]", nil
	}
	encoded, err := json.Marshal(keywords)
	if err != nil {
		return "", fmt.Errorf("packs: encode keywords: %w", err)
	}
	return string(encoded), nil
}

func isVersionConflict(err error) bool {
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") && strings.Contains(message, "pack_versions")
}
