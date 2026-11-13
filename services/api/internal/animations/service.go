package animations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/google/uuid"
)

const maxMotionTextLength = 500

// selectAnimation reads drafts with the checksum of the static sticker now at
// their index; a missing sticker yields an empty checksum and therefore stale.
const selectAnimation = `SELECT a.id, a.pack_id, a.sticker_index, a.static_checksum_sha256, a.motion_text,
	a.clip_byte_size, a.status, a.use_animated, a.animated_path, a.animated_checksum_sha256,
	a.frames, a.fps, a.duration_ms, a.quality, a.width, a.height, a.animated_byte_size,
	a.error_code, a.error_message, COALESCE(s.checksum_sha256, '')
	FROM sticker_animations a
	JOIN packs p ON p.id = a.pack_id
	LEFT JOIN stickers s ON s.pack_id = a.pack_id AND s.version = p.current_version AND s.sort_order = a.sticker_index`

// Service manages animation drafts on the single factory database connection.
type Service struct {
	db    *sql.DB
	files *assets.Store

	// Clip worker state: wake is signalled after an upload commits; timeout
	// bounds one clip child.
	wake        chan struct{}
	timeout     time.Duration
	recoverOnce sync.Once
	recoverErr  error
}

// NewService wires drafts to the database and the data-root file store.
func NewService(db *sql.DB, files *assets.Store) *Service {
	return &Service{db: db, files: files, wake: make(chan struct{}, 1), timeout: ClipTimeout}
}

// Upload stores a clip for sticker index of the pack's current version and
// queues it, replacing any existing draft for that sticker. Old files are
// removed only after the replacement commits.
func (s *Service) Upload(ctx context.Context, packID string, index int, clip io.Reader, filename, motionText string) (Animation, error) {
	motionText = strings.TrimSpace(motionText)
	if !isUUID(packID) {
		return Animation{}, ErrNotFound
	}
	if index < 0 || utf8.RuneCountInString(motionText) > maxMotionTextLength {
		return Animation{}, ErrInvalid
	}
	// Cheap pre-check so a refused upload never touches disk; repeated in the tx.
	if _, err := uploadTarget(ctx, s.db, packID, index); err != nil {
		return Animation{}, err
	}
	saved, err := s.files.SaveClip(ctx, clip, filename)
	if err != nil {
		if errors.Is(err, assets.ErrInvalidClip) || errors.Is(err, assets.ErrClipTooLarge) || errors.Is(err, assets.ErrInvalidFilename) {
			return Animation{}, fmt.Errorf("%w: %w", ErrInvalid, err)
		}
		return Animation{}, fmt.Errorf("animations: save clip: %w", err)
	}
	old, err := s.replace(ctx, packID, index, motionText, saved)
	if err != nil {
		_ = s.files.RemoveClip(saved.ID)
		return Animation{}, err
	}
	s.signal()
	if old.id != "" {
		_ = s.files.RemoveClip(old.clipID)
		_ = s.files.RemoveAnimated(old.id)
	}
	return s.get(ctx, packID, index)
}

type previous struct{ id, clipID string }

func (s *Service) replace(ctx context.Context, packID string, index int, motionText string, clip assets.Clip) (previous, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return previous{}, fmt.Errorf("animations: begin upload: %w", err)
	}
	defer tx.Rollback()
	staticChecksum, err := uploadTarget(ctx, tx, packID, index)
	if err != nil {
		return previous{}, err
	}
	var old previous
	var status string
	err = tx.QueryRowContext(ctx, `SELECT id, clip_id, status FROM sticker_animations
		WHERE pack_id = ? AND sticker_index = ?`, packID, index).Scan(&old.id, &old.clipID, &status)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return previous{}, fmt.Errorf("animations: read draft: %w", err)
	case Status(status) == StatusRunning:
		// The worker is reading this clip; replacing it now would orphan its output.
		return previous{}, ErrJobActive
	default:
		if _, err := tx.ExecContext(ctx, `DELETE FROM sticker_animations WHERE id = ?`, old.id); err != nil {
			return previous{}, fmt.Errorf("animations: delete old draft: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sticker_animations
		(id, pack_id, sticker_index, static_checksum_sha256, motion_text, clip_id, clip_path, clip_byte_size, clip_checksum_sha256, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), packID, index, staticChecksum, motionText, clip.ID, clip.RelativePath,
		clip.ByteSize, clip.ChecksumSHA256, string(StatusQueued)); err != nil {
		return previous{}, fmt.Errorf("animations: insert draft: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return previous{}, fmt.Errorf("animations: commit upload: %w", err)
	}
	return old, nil
}

// List returns the pack's drafts by sticker index with Stale computed against
// the current version.
func (s *Service) List(ctx context.Context, packID string) ([]Animation, error) {
	if err := packExists(ctx, s.db, packID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, selectAnimation+` WHERE a.pack_id = ? ORDER BY a.sticker_index`, packID)
	if err != nil {
		return nil, fmt.Errorf("animations: list: %w", err)
	}
	defer rows.Close()
	items := []Animation{}
	for rows.Next() {
		item, err := scanAnimation(rows)
		if err != nil {
			return nil, fmt.Errorf("animations: scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("animations: read list: %w", err)
	}
	return items, nil
}

// SetUse toggles whether a ready draft goes into the next publish.
func (s *Service) SetUse(ctx context.Context, packID string, index int, use bool) (Animation, error) {
	if !isUUID(packID) {
		return Animation{}, ErrNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Animation{}, fmt.Errorf("animations: begin toggle: %w", err)
	}
	defer tx.Rollback()
	if err := checkNoJob(ctx, tx, packID); err != nil {
		return Animation{}, err
	}
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM sticker_animations WHERE pack_id = ? AND sticker_index = ?`,
		packID, index).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return Animation{}, ErrNotFound
	}
	if err != nil {
		return Animation{}, fmt.Errorf("animations: read draft: %w", err)
	}
	if Status(status) != StatusReady {
		return Animation{}, ErrNotReady
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sticker_animations SET use_animated = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE pack_id = ? AND sticker_index = ?`,
		use, packID, index); err != nil {
		return Animation{}, fmt.Errorf("animations: toggle: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Animation{}, fmt.Errorf("animations: commit toggle: %w", err)
	}
	return s.get(ctx, packID, index)
}

// Delete removes a draft and then its clip and animated files.
func (s *Service) Delete(ctx context.Context, packID string, index int) error {
	if !isUUID(packID) {
		return ErrNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("animations: begin delete: %w", err)
	}
	defer tx.Rollback()
	if err := checkNoJob(ctx, tx, packID); err != nil {
		return err
	}
	var old previous
	var status string
	err = tx.QueryRowContext(ctx, `SELECT id, clip_id, status FROM sticker_animations
		WHERE pack_id = ? AND sticker_index = ?`, packID, index).Scan(&old.id, &old.clipID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("animations: read draft: %w", err)
	}
	if Status(status) == StatusRunning {
		return ErrJobActive
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sticker_animations WHERE id = ?`, old.id); err != nil {
		return fmt.Errorf("animations: delete draft: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("animations: commit delete: %w", err)
	}
	return errors.Join(s.files.RemoveClip(old.clipID), s.files.RemoveAnimated(old.id))
}

// Busy reports whether any draft of the pack is queued or running.
func (s *Service) Busy(ctx context.Context, packID string) (bool, error) {
	var busy bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM sticker_animations
		WHERE pack_id = ? AND status IN ('queued', 'running'))`, packID).Scan(&busy); err != nil {
		return false, fmt.Errorf("animations: check busy: %w", err)
	}
	return busy, nil
}

// Publishable returns the drafts a publish job copies: ready, enabled and not
// stale against the current version, by sticker index.
func (s *Service) Publishable(ctx context.Context, packID string) ([]jobs.PublishableAnimation, error) {
	rows, err := s.db.QueryContext(ctx, selectAnimation+` WHERE a.pack_id = ? AND a.status = 'ready'
		AND a.use_animated = 1 ORDER BY a.sticker_index`, packID)
	if err != nil {
		return nil, fmt.Errorf("animations: list publishable: %w", err)
	}
	defer rows.Close()
	items := []jobs.PublishableAnimation{}
	for rows.Next() {
		item, err := scanAnimation(rows)
		if err != nil {
			return nil, fmt.Errorf("animations: scan: %w", err)
		}
		if item.Stale {
			continue
		}
		items = append(items, jobs.PublishableAnimation{
			StickerIndex: item.StickerIndex, Path: item.AnimatedPath, Checksum: item.AnimatedChecksum,
			Frames: item.Frames, DurationMS: item.DurationMS, Width: item.Width, Height: item.Height,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("animations: list publishable: %w", err)
	}
	return items, nil
}

func (s *Service) get(ctx context.Context, packID string, index int) (Animation, error) {
	item, err := scanAnimation(s.db.QueryRowContext(ctx, selectAnimation+` WHERE a.pack_id = ? AND a.sticker_index = ?`, packID, index))
	if errors.Is(err, sql.ErrNoRows) {
		return Animation{}, ErrNotFound
	}
	if err != nil {
		return Animation{}, fmt.Errorf("animations: get: %w", err)
	}
	return item, nil
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// uploadTarget checks that a clip may be attached to sticker index and
// returns the static checksum it is drafted against.
func uploadTarget(ctx context.Context, q querier, packID string, index int) (string, error) {
	var status string
	var count sql.NullInt64
	var checksum sql.NullString
	err := q.QueryRowContext(ctx, `SELECT p.status, v.sticker_count, s.checksum_sha256 FROM packs p
		LEFT JOIN pack_versions v ON v.pack_id = p.id AND v.version = p.current_version
		LEFT JOIN stickers s ON s.pack_id = p.id AND s.version = p.current_version AND s.sort_order = ?
		WHERE p.id = ?`, index, packID).Scan(&status, &count, &checksum)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("animations: read pack: %w", err)
	}
	if status == "revoked" {
		return "", fmt.Errorf("%w: pack is revoked", ErrInvalid)
	}
	if err := checkNoJob(ctx, q, packID); err != nil {
		return "", err
	}
	if !count.Valid || int64(index) >= count.Int64 || !checksum.Valid {
		return "", fmt.Errorf("%w: sticker index out of range", ErrInvalid)
	}
	return checksum.String, nil
}

// checkNoJob refuses draft changes while a sheet or publish job is queued or
// running: either will commit a new version under the draft.
func checkNoJob(ctx context.Context, q querier, packID string) error {
	var active bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM processing_jobs
		WHERE pack_id = ? AND status IN ('queued', 'running'))`, packID).Scan(&active); err != nil {
		return fmt.Errorf("animations: check active job: %w", err)
	}
	if active {
		return ErrJobActive
	}
	return nil
}

func packExists(ctx context.Context, q querier, packID string) error {
	if !isUUID(packID) {
		return ErrNotFound
	}
	var one int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM packs WHERE id = ?`, packID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("animations: read pack: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAnimation(row scanner) (Animation, error) {
	var item Animation
	var status, current string
	if err := row.Scan(&item.ID, &item.PackID, &item.StickerIndex, &item.StaticChecksum, &item.MotionText,
		&item.ClipByteSize, &status, &item.UseAnimated, &item.AnimatedPath, &item.AnimatedChecksum,
		&item.Frames, &item.FPS, &item.DurationMS, &item.Quality, &item.Width, &item.Height, &item.AnimatedByteSize,
		&item.ErrorCode, &item.ErrorMessage, &current); err != nil {
		return Animation{}, err
	}
	item.Status = Status(status)
	item.Stale = current != item.StaticChecksum
	return item, nil
}

func isUUID(value string) bool {
	return uuid.Validate(value) == nil
}
