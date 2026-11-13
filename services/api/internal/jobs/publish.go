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
	"slices"
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
// against is pinned and re-checked inside the transaction, and the transaction
// re-derives the publishable drafts with the same filters; a draft, toggle or
// static that changed in between is a conflict instead of a silently different
// publication. The request freezes the revalidated list.
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
	// The pre-read list is only a proposal: re-derive it on this transaction,
	// which is the point where no draft writer can slip in (they all refuse
	// while a job is active, and the job row does not exist yet).
	busyNow, err := animationBusy(ctx, tx, packID)
	if err != nil {
		return Job{}, err
	}
	validated, err := publishableAnimations(ctx, tx, packID, current)
	if err != nil {
		return Job{}, err
	}
	published, count, err := publishedAnimations(ctx, tx, packID, current)
	if err != nil {
		return Job{}, err
	}
	if err := checkAnimations(animations, count); err != nil {
		return Job{}, err
	}
	if busyNow || !slices.Equal(animations, validated) {
		// A clip or draft changed after the drafts were read: publishing now
		// would freeze a set the request never approved.
		return Job{}, ErrAnimationBusy
	}
	if sameAnimations(validated, published) {
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
		Animations:    validated,
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

// animationBusy reports whether any draft of the pack is queued or running,
// read on the caller's transaction.
func animationBusy(ctx context.Context, tx *sql.Tx, packID string) (bool, error) {
	var busy bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM sticker_animations
		WHERE pack_id = ? AND status IN ('queued', 'running'))`, packID).Scan(&busy); err != nil {
		return false, fmt.Errorf("jobs: check animation busy: %w", err)
	}
	return busy, nil
}

// publishableAnimations re-derives the drafts a publish job may copy, with the
// same filters as animations.Publishable (the service owns the row, not this
// package): ready, enabled and drafted against the current version's static
// sticker at their index.
func publishableAnimations(ctx context.Context, tx *sql.Tx, packID string, version int) ([]PublishableAnimation, error) {
	rows, err := tx.QueryContext(ctx, `SELECT a.sticker_index, a.animated_path, a.animated_checksum_sha256,
		a.frames, a.duration_ms, a.width, a.height
		FROM sticker_animations a
		JOIN stickers s ON s.pack_id = a.pack_id AND s.version = ? AND s.sort_order = a.sticker_index
		WHERE a.pack_id = ? AND a.status = 'ready' AND a.use_animated = 1
			AND a.static_checksum_sha256 = s.checksum_sha256
		ORDER BY a.sticker_index`, version, packID)
	if err != nil {
		return nil, fmt.Errorf("jobs: read publishable animations: %w", err)
	}
	defer rows.Close()
	items := []PublishableAnimation{}
	for rows.Next() {
		var item PublishableAnimation
		if err := rows.Scan(&item.StickerIndex, &item.Path, &item.Checksum, &item.Frames,
			&item.DurationMS, &item.Width, &item.Height); err != nil {
			return nil, fmt.Errorf("jobs: scan animation: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: read publishable animations: %w", err)
	}
	return items, nil
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
