package jobs

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/google/uuid"
)

// EnqueuePublish queues a Go-only job that republishes the current version
// with the pack's publishable animated drafts as the next version.
//
// The AnimationSource shares the single SQLite connection, so Busy and
// Publishable run before the transaction (calling them inside it would wait on
// the connection the transaction holds). The version they were computed
// against is pinned and re-checked inside the transaction, and the list is
// frozen into the request, so the job publishes exactly what was checked.
func (r *Runner) EnqueuePublish(ctx context.Context, packID string) (Job, error) {
	if !isUUID(packID) {
		return Job{}, ErrNotFound
	}
	if r.animations == nil {
		return Job{}, fmt.Errorf("%w: animations are not configured", ErrInvalid)
	}
	var base int
	err := r.db.QueryRowContext(ctx, `SELECT current_version FROM packs WHERE id = ?`, packID).Scan(&base)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("jobs: read pack: %w", err)
	}
	busy, err := r.animations.Busy(ctx, packID)
	if err != nil {
		return Job{}, err
	}
	animations, err := r.animations.Publishable(ctx, packID)
	if err != nil {
		return Job{}, err
	}
	sort.Slice(animations, func(i, j int) bool { return animations[i].StickerIndex < animations[j].StickerIndex })

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, err
	}
	defer tx.Rollback()

	snapshot, err := loadEnqueueSnapshot(ctx, tx, packID)
	if err != nil {
		return Job{}, err
	}
	if snapshot.status == string(packs.StatusRevoked) {
		return Job{}, ErrPackRevoked
	}
	var current int
	if err := tx.QueryRowContext(ctx, `SELECT current_version FROM packs WHERE id = ?`, packID).Scan(&current); err != nil {
		return Job{}, fmt.Errorf("jobs: read pack: %w", err)
	}
	if current < 1 {
		return Job{}, fmt.Errorf("%w: pack has no published version", ErrNotFound)
	}
	active, err := hasActiveJob(ctx, tx, packID)
	if err != nil {
		return Job{}, err
	}
	// A moved version means a job committed after the drafts were read.
	if active || current != base {
		return Job{}, ErrJobActive
	}
	if busy {
		return Job{}, ErrAnimationBusy
	}
	published, count, err := publishedAnimations(ctx, tx, packID, current)
	if err != nil {
		return Job{}, err
	}
	if err := checkAnimations(animations, count); err != nil {
		return Job{}, err
	}
	if sameAnimations(animations, published) {
		return Job{}, ErrNoChanges
	}

	var version int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM (
			SELECT version FROM processing_jobs WHERE pack_id = ?
			UNION ALL
			SELECT version FROM pack_versions WHERE pack_id = ?
		)`, packID, packID).Scan(&version); err != nil {
		return Job{}, fmt.Errorf("jobs: reserve version: %w", err)
	}
	id := uuid.NewString()
	payload, err := json.Marshal(JobRequest{
		SchemaVersion: RequestSchemaVersion,
		Kind:          KindPublish,
		JobID:         id,
		PackID:        packID,
		Version:       version,
		Catalog:       snapshot.catalog,
		BaseVersion:   current,
		Animations:    animations,
	})
	if err != nil {
		return Job{}, fmt.Errorf("jobs: encode request: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO processing_jobs (id, pack_id, version, status, request_json)
		VALUES (?, ?, ?, 'queued', ?)`, id, packID, version, string(payload)); err != nil {
		if isActiveJobConflict(err) {
			return Job{}, ErrJobActive
		}
		return Job{}, fmt.Errorf("jobs: insert job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Job{}, err
	}
	r.signal()
	return r.Get(ctx, id)
}

// publishedAnimations returns sticker index -> animated checksum of the
// version's animated stickers, plus its sticker count.
func publishedAnimations(ctx context.Context, tx *sql.Tx, packID string, version int) (map[int]string, int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT sort_order, animated_checksum_sha256 FROM stickers
		WHERE pack_id = ? AND version = ?`, packID, version)
	if err != nil {
		return nil, 0, fmt.Errorf("jobs: read stickers: %w", err)
	}
	defer rows.Close()
	published := map[int]string{}
	count := 0
	for rows.Next() {
		var index int
		var checksum string
		if err := rows.Scan(&index, &checksum); err != nil {
			return nil, 0, fmt.Errorf("jobs: scan sticker: %w", err)
		}
		count++
		if checksum != "" {
			published[index] = checksum
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("jobs: read stickers: %w", err)
	}
	return published, count, nil
}

// checkAnimations rejects a draft list that cannot belong to the version; the
// list is sorted by index, so a duplicate is adjacent.
func checkAnimations(animations []PublishableAnimation, count int) error {
	for i, a := range animations {
		if a.StickerIndex < 0 || a.StickerIndex >= count || (i > 0 && animations[i-1].StickerIndex == a.StickerIndex) {
			return fmt.Errorf("%w: animation for sticker %d", ErrInvalid, a.StickerIndex)
		}
	}
	return nil
}

func sameAnimations(animations []PublishableAnimation, published map[int]string) bool {
	if len(animations) != len(published) {
		return false
	}
	for _, a := range animations {
		if published[a.StickerIndex] != a.Checksum {
			return false
		}
	}
	return true
}

// preparePublish fills the empty job output directory with the base
// version's statics and thumbs plus the frozen animated drafts, and returns
// the manifest for the existing stage/commit path. StageBundle re-verifies
// every file; the animated checksums are verified here against the request.
func (r *Runner) preparePublish(ctx context.Context, request JobRequest) (assets.PackManifest, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT filename, object_path, thumb_path, width, height, emoji_hint, keywords_json
		FROM stickers WHERE pack_id = ? AND version = ? ORDER BY sort_order`, request.PackID, request.BaseVersion)
	if err != nil {
		return assets.PackManifest{}, fmt.Errorf("read base stickers: %w", err)
	}
	type baseSticker struct {
		sticker               assets.ManifestSticker
		objectPath, thumbPath string
	}
	var base []baseSticker
	for rows.Next() {
		var item baseSticker
		var keywords string
		if err := rows.Scan(&item.sticker.File, &item.objectPath, &item.thumbPath, &item.sticker.Width,
			&item.sticker.Height, &item.sticker.EmojiHint, &keywords); err != nil {
			rows.Close()
			return assets.PackManifest{}, fmt.Errorf("scan base sticker: %w", err)
		}
		if err := json.Unmarshal([]byte(keywords), &item.sticker.Keywords); err != nil {
			rows.Close()
			return assets.PackManifest{}, fmt.Errorf("base sticker keywords: %w", err)
		}
		item.sticker.ThumbFile = path.Base(item.thumbPath)
		base = append(base, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return assets.PackManifest{}, fmt.Errorf("read base stickers: %w", err)
	}
	if len(base) == 0 {
		return assets.PackManifest{}, fmt.Errorf("%w: base version %d has no stickers", ErrInvalid, request.BaseVersion)
	}

	output := filepath.Join(request.WorkDir, "output")
	manifest := assets.PackManifest{
		PackID:      request.PackID,
		Slug:        request.Catalog.Slug,
		Name:        request.Catalog.Name,
		Author:      request.Catalog.Author,
		Description: request.Catalog.Description,
		Tags:        append([]string(nil), request.Catalog.Tags...),
		Version:     request.Version,
		Stickers:    make([]assets.ManifestSticker, len(base)),
	}
	for i, item := range base {
		if err := ctx.Err(); err != nil {
			return assets.PackManifest{}, err
		}
		for relative, name := range map[string]string{item.objectPath: item.sticker.File, item.thumbPath: item.sticker.ThumbFile} {
			if _, err := copyOut(r.files.OpenCommitted, relative, filepath.Join(output, filepath.Base(name))); err != nil {
				return assets.PackManifest{}, err
			}
		}
		manifest.Stickers[i] = item.sticker
	}
	if len(request.Animations) > 0 {
		if err := os.Mkdir(filepath.Join(output, "animated"), 0o755); err != nil {
			return assets.PackManifest{}, fmt.Errorf("create animated output: %w", err)
		}
	}
	for _, a := range request.Animations {
		if err := ctx.Err(); err != nil {
			return assets.PackManifest{}, err
		}
		if a.StickerIndex < 0 || a.StickerIndex >= len(base) {
			return assets.PackManifest{}, fmt.Errorf("%w: animation for sticker %d", ErrInvalid, a.StickerIndex)
		}
		file := manifest.Stickers[a.StickerIndex].File
		sum, err := copyOut(r.files.OpenAnimated, a.Path, filepath.Join(output, "animated", filepath.Base(file)))
		if err != nil {
			return assets.PackManifest{}, err
		}
		if sum != a.Checksum {
			return assets.PackManifest{}, fmt.Errorf("animation for sticker %d changed after it was queued", a.StickerIndex)
		}
		manifest.Stickers[a.StickerIndex].Animated = &assets.ManifestAnimated{
			File: file, Frames: a.Frames, DurationMS: a.DurationMS, Width: a.Width, Height: a.Height,
		}
	}
	return manifest, nil
}

// copyOut copies a file opened through a store helper (no symlink following)
// to a new file dst and returns the SHA256 hex of the copied bytes.
func copyOut(open func(string) (*os.File, error), relative, dst string) (string, error) {
	src, err := open(relative)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", relative, err)
	}
	defer src.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", filepath.Base(dst), err)
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), src); err != nil {
		out.Close()
		return "", fmt.Errorf("copy %s: %w", relative, err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("copy %s: %w", relative, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
