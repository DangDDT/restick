package animations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"golang.org/x/image/webp"
)

const (
	// ClipTimeout bounds one clip child. It reuses the sheet child's bound: a
	// 4 s clip is at most 48 frames at about 0.3 s of cutout each plus a
	// four-step encode ladder, far below it, while a hung decoder is still killed.
	ClipTimeout = jobs.DefaultTimeout

	clipFPS     = 12
	clipQuality = 75
	// maxAnimatedBytes is the clip_job size ceiling; a larger file is not its output.
	maxAnimatedBytes = 512_000
)

// Error codes stored in sticker_animations.error_code besides the machine
// codes reported by clip_job; the shared ones keep the jobs spelling.
const (
	CodeInvalidRequest = "invalid_request"
	CodeOutputFailed   = "output_failed"
)

type claimedClip struct {
	id, packID, clipID string
	index              int
}

// RunWorker runs the clip queue until ctx is cancelled. Startup recovery runs
// first. childLock is the sheet runner's child lock, held across claim and
// child so there is never more than one Python child and a clip waiting for
// the lock stays queued (editable) instead of running.
func (s *Service) RunWorker(ctx context.Context, p ClipProcessor, childLock *sync.Mutex) error {
	if err := s.Recover(ctx); err != nil {
		return err
	}
	for {
		childLock.Lock()
		item, err := s.claim(ctx)
		if err == nil && item != nil {
			s.execute(ctx, p, *item)
		}
		childLock.Unlock()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if item != nil {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-s.wake:
		}
	}
}

func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// claim promotes the oldest queued draft to running in one short transaction.
func (s *Service) claim(ctx context.Context) (*claimedClip, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var item claimedClip
	err = tx.QueryRowContext(ctx, `SELECT id, pack_id, sticker_index, clip_id FROM sticker_animations
		WHERE status = 'queued' ORDER BY created_at, rowid LIMIT 1`).Scan(&item.id, &item.packID, &item.index, &item.clipID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("animations: claim: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE sticker_animations SET status = 'running',
		started_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), finished_at = NULL,
		error_code = '', error_message = '', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND status = 'queued'`, item.id)
	if err != nil {
		return nil, fmt.Errorf("animations: claim: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return nil, fmt.Errorf("animations: claim: %w", err)
	} else if affected == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &item, nil
}

// execute runs one claimed clip; the caller holds the child lock.
func (s *Service) execute(ctx context.Context, p ClipProcessor, item claimedClip) {
	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	request, workDir, code, err := s.prepare(runCtx, item)
	if workDir != "" {
		defer os.RemoveAll(workDir)
	}
	if err != nil {
		s.failRun(ctx, runCtx, item, code, err)
		return
	}
	result, err := p.ProcessClip(runCtx, request)
	if err != nil {
		var processErr *jobs.ProcessError
		code := jobs.CodeProcessorFailed
		if errors.As(err, &processErr) && processErr.Code != "" {
			code = processErr.Code
		}
		s.failRun(ctx, runCtx, item, code, err)
		return
	}
	data, err := readAnimated(request, result)
	if err != nil {
		s.fail(ctx, item, jobs.CodeResultInvalid, err.Error())
		return
	}
	relative, err := s.files.SaveAnimated(item.id, data)
	if err != nil {
		s.fail(ctx, item, CodeOutputFailed, err.Error())
		return
	}
	sum := sha256.Sum256(data)
	// result.Diagnostics has no column on the draft and is dropped here.
	written, err := s.update(ctx, item.id, `status = 'ready', animated_path = ?, animated_checksum_sha256 = ?,
		animated_byte_size = ?, frames = ?, fps = ?, duration_ms = ?, quality = ?, width = ?, height = ?,
		error_code = '', error_message = ''`,
		relative, hex.EncodeToString(sum[:]), len(data), result.Frames, result.FPS, result.DurationMS,
		result.Quality, result.Width, result.Height)
	if err != nil || !written {
		// The row was replaced, deleted or interrupted meanwhile: the file is no one's.
		_ = s.files.RemoveAnimated(item.id)
	}
}

// prepare resolves the outline exactly like a sheet job (the pack's design,
// no defaults) and gives the child an empty output directory.
func (s *Service) prepare(ctx context.Context, item claimedClip) (ClipRequest, string, string, error) {
	var designID, color sql.NullString
	var width sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT p.design_id, d.outline_width, d.outline_color
		FROM packs p LEFT JOIN designs d ON d.id = p.design_id WHERE p.id = ?`, item.packID).
		Scan(&designID, &width, &color)
	if err != nil {
		return ClipRequest{}, "", jobs.CodeProcessorFailed, fmt.Errorf("read pack design: %w", err)
	}
	if !designID.Valid || designID.String == "" || !width.Valid {
		return ClipRequest{}, "", CodeInvalidRequest, errors.New("pack has no design")
	}
	clipPath, err := s.files.ClipPath(item.clipID)
	if err != nil {
		return ClipRequest{}, "", CodeInvalidRequest, fmt.Errorf("resolve clip: %w", err)
	}
	workDir, err := s.files.WorkDir("clip-" + item.id)
	if err != nil {
		return ClipRequest{}, "", jobs.CodeProcessorFailed, err
	}
	outputDir := filepath.Join(workDir, "output")
	if err := os.RemoveAll(outputDir); err != nil {
		return ClipRequest{}, workDir, jobs.CodeProcessorFailed, fmt.Errorf("clear clip output: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return ClipRequest{}, workDir, jobs.CodeProcessorFailed, fmt.Errorf("create clip output: %w", err)
	}
	return ClipRequest{
		AnimationID: item.id, PackID: item.packID, StickerIndex: item.index, ClipPath: clipPath,
		OutputDir: outputDir, FPS: clipFPS, Quality: clipQuality,
		OutlineWidth: int(width.Int64), OutlineColor: color.String,
	}, workDir, "", nil
}

// readAnimated treats the result as a claim: the file must be the fixed name
// for this index, a regular file under the ceiling, 512x512 by DecodeConfig
// (Decode fails on animated WebP) and match the reported size.
func readAnimated(r ClipRequest, result ClipResult) ([]byte, error) {
	want := fmt.Sprintf("animated/%03d.webp", r.StickerIndex)
	switch {
	case result.Filename != want:
		return nil, fmt.Errorf("result file %q is not %q", result.Filename, want)
	case result.Width != assets.StickerDimension || result.Height != assets.StickerDimension:
		return nil, fmt.Errorf("result is %dx%d", result.Width, result.Height)
	case result.Frames < 2 || result.FPS < 1 || result.DurationMS < 1 || result.Quality < 1:
		return nil, fmt.Errorf("result reports %d frames at %d fps, %d ms, quality %d",
			result.Frames, result.FPS, result.DurationMS, result.Quality)
	}
	file, err := os.OpenFile(filepath.Join(r.OutputDir, filepath.FromSlash(want)), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("open result file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("result file is not a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxAnimatedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read result file: %w", err)
	}
	if len(data) > maxAnimatedBytes {
		return nil, fmt.Errorf("result file exceeds %d bytes", maxAnimatedBytes)
	}
	if len(data) != result.ByteSize {
		return nil, fmt.Errorf("result file has %d bytes, result reports %d", len(data), result.ByteSize)
	}
	cfg, err := webp.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("result file is not WebP: %w", err)
	}
	if cfg.Width != assets.StickerDimension || cfg.Height != assets.StickerDimension {
		return nil, fmt.Errorf("result file is %dx%d", cfg.Width, cfg.Height)
	}
	return data, nil
}

// failRun maps a run error: a timeout fails, a shutdown interrupts (requeued
// on the next start), anything else fails with code.
func (s *Service) failRun(ctx, runCtx context.Context, item claimedClip, code string, err error) {
	switch {
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		s.fail(ctx, item, jobs.CodeTimeout, fmt.Sprintf("processing did not finish within %s", s.timeout))
	case ctx.Err() != nil:
		_, _ = s.update(ctx, item.id, `status = 'interrupted', error_code = ?, error_message = ?`,
			jobs.CodeShutdown, "the server stopped while this clip was running")
	default:
		s.fail(ctx, item, code, err.Error())
	}
}

func (s *Service) fail(ctx context.Context, item claimedClip, code, message string) {
	// A file from an earlier crashed attempt must not outlive a failure.
	_ = s.files.RemoveAnimated(item.id)
	_, _ = s.update(ctx, item.id, `status = 'failed', error_code = ?, error_message = ?`,
		code, truncate(message, maxMessageChars))
}

// update writes a terminal state only while this exact row is still running,
// so a late write never touches a replaced, deleted or interrupted row.
func (s *Service) update(ctx context.Context, id, set string, args ...any) (bool, error) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(writeCtx, `UPDATE sticker_animations SET `+set+`,
		finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND status = 'running'`, append(args, id)...)
	if err != nil {
		return false, fmt.Errorf("animations: finish clip: %w", err)
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

// Recover is the idempotent startup step: drafts left running become terminal
// interrupted/startup_recovery like sheet jobs (a crash is not evidence of
// completion, and retrying a clip that crashes the server would loop), only
// rows interrupted by a graceful shutdown are queued again, and regular files
// in clips/ and animations/ that no draft references are removed. It must run
// before the HTTP server starts: an upload writes its clip before its row.
func (s *Service) Recover(ctx context.Context) error {
	s.recoverOnce.Do(func() { s.recoverErr = s.recover(ctx) })
	return s.recoverErr
}

func (s *Service) recover(ctx context.Context) error {
	// A dead process is never evidence of completion.
	if _, err := s.db.ExecContext(ctx, `UPDATE sticker_animations SET status = 'interrupted',
		error_code = ?, error_message = ?,
		finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE status = 'running'`, jobs.CodeStartupRecovery, "the server stopped while this clip was running"); err != nil {
		return fmt.Errorf("animations: interrupt stale clips: %w", err)
	}
	// Only a graceful shutdown proves the clip was interrupted, not the clip
	// itself: anything else stays terminal so the owner can reupload it.
	if _, err := s.db.ExecContext(ctx, `UPDATE sticker_animations SET status = 'queued',
		error_code = '', error_message = '', started_at = NULL, finished_at = NULL,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE status = 'interrupted' AND error_code = ?`, jobs.CodeShutdown); err != nil {
		return fmt.Errorf("animations: requeue clips: %w", err)
	}
	s.pruneOrphans(ctx)
	return nil
}

// pruneOrphans removes draft files left by crashes. Unreferenced files are
// never public, so a failure is ignored rather than blocking startup.
func (s *Service) pruneOrphans(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, clip_id FROM sticker_animations`)
	if err != nil {
		return
	}
	ids, clips := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var id, clipID string
		if err := rows.Scan(&id, &clipID); err != nil {
			rows.Close()
			return
		}
		ids[id], clips[clipID] = true, true
	}
	rows.Close()
	if rows.Err() != nil {
		return // never prune against a partial keep-set
	}
	_ = s.files.PruneDrafts(func(dir, name string) bool {
		if dir == "clips" {
			clipID, _, _ := strings.Cut(name, ".")
			return clips[clipID]
		}
		id, ok := strings.CutSuffix(name, ".webp")
		return ok && ids[id]
	})
}
