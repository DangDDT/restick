package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/studio"
	"github.com/google/uuid"
)

const (
	timestampLayout = "2006-01-02T15:04:05Z"
	maxMessageChars = 1000
	maxDiagnostics  = 4000
	jobSelect       = `SELECT id, pack_id, version, status, request_json, result_json, error_code, error_message, diagnostics, created_at, started_at, finished_at FROM processing_jobs`
)

// Runner is the single durable worker: it claims one queued job at a time,
// runs exactly one processor child and publishes through packs. Transactions are
// only held for the short claim/commit writes, never while the child runs.
type Runner struct {
	db        *sql.DB
	files     *assets.Store
	processor Processor
	timeout   time.Duration
	wake      chan struct{}

	// childMu serializes processor runs so there is never more than one child.
	childMu sync.Mutex

	mu           sync.Mutex
	activePack   string
	cancelActive context.CancelFunc

	recoverOnce sync.Once
	recoverErr  error
}

// NewRunner builds the worker for the single factory instance.
func NewRunner(db *sql.DB, files *assets.Store, processor Processor, timeout time.Duration) *Runner {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Runner{
		db:        db,
		files:     files,
		processor: processor,
		timeout:   timeout,
		wake:      make(chan struct{}, 1),
	}
}

// Enqueue snapshots the pack, design, source and catalog and reserves the next
// version in one short transaction. Only one job per pack may be queued or
// running, enforced by the database.
func (r *Runner) Enqueue(ctx context.Context, packID, sourceID, designID string) (Job, error) {
	if !isUUID(packID) {
		return Job{}, ErrNotFound
	}
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
	source, err := snapshot.sourceRef(sourceID)
	if err != nil {
		return Job{}, err
	}
	design, err := snapshot.designRef(designID)
	if err != nil {
		return Job{}, err
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
	request := JobRequest{
		SchemaVersion: RequestSchemaVersion,
		JobID:         id,
		PackID:        packID,
		Version:       version,
		DesignID:      design,
		Design:        snapshot.design,
		Source:        source,
		Catalog:       snapshot.catalog,
		Rows:          snapshot.rows,
		Columns:       snapshot.columns,
		OutlineWidth:  snapshot.outlineWidth,
		OutlineColor:  snapshot.outlineColor,
	}
	payload, err := json.Marshal(request)
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

// Get returns one job row in any state.
func (r *Runner) Get(ctx context.Context, id string) (Job, error) {
	if !isUUID(id) {
		return Job{}, ErrNotFound
	}
	job, err := scanJob(r.db.QueryRowContext(ctx, jobSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("jobs: get job: %w", err)
	}
	return job, nil
}

// LatestForPack returns the latest attempt, including terminal diagnostics.
// Reserved versions increase even when an attempt fails or is interrupted.
func (r *Runner) LatestForPack(ctx context.Context, packID string) (*Job, error) {
	if !isUUID(packID) {
		return nil, ErrNotFound
	}
	job, err := scanJob(r.db.QueryRowContext(ctx, jobSelect+` WHERE pack_id = ? ORDER BY version DESC LIMIT 1`, packID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jobs: latest pack job: %w", err)
	}
	return &job, nil
}

// Revoke ends a pack in one transaction: it writes the single tombstone,
// interrupts every queued or running job and then cancels the active child.
// A late completion can never publish because packs.CommitProcessed requires
// the job to still be running and the pack to be live.
func (r *Runner) Revoke(ctx context.Context, packID string) error {
	if !isUUID(packID) {
		return ErrNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	var currentVersion int
	err = tx.QueryRowContext(ctx, `SELECT status, current_version FROM packs WHERE id = ?`, packID).
		Scan(&status, &currentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("jobs: read pack: %w", err)
	}
	if status == string(packs.StatusRevoked) {
		return nil
	}
	// A draft pack with a queued or running job is cancelled by shutdown or by
	// waiting for the job; it is not revocable (spec §6).
	if status == string(packs.StatusDraft) {
		active, err := hasActiveJob(ctx, tx, packID)
		if err != nil {
			return err
		}
		if active {
			return ErrJobActive
		}
	}
	if currentVersion < 1 {
		return fmt.Errorf("%w: pack has no committed version to revoke", ErrInvalid)
	}
	var checksum string
	if err := tx.QueryRowContext(ctx, `SELECT checksum_sha256 FROM pack_versions WHERE pack_id = ? AND version = ?`,
		packID, currentVersion).Scan(&checksum); err != nil {
		return fmt.Errorf("jobs: read current version: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE packs SET status = 'revoked',
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ? AND status != 'revoked'`, packID); err != nil {
		return fmt.Errorf("jobs: revoke pack: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE processing_jobs SET status = 'interrupted', error_code = ?, error_message = ?,
		finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE pack_id = ? AND status IN ('queued', 'running')`,
		CodePackRevoked, "pack was revoked before the job finished", packID); err != nil {
		return fmt.Errorf("jobs: interrupt pack jobs: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO pack_tombstones (pack_id, version, revoked_at, reason_code, checksum_sha256)
		VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), 'revoked', ?)
		ON CONFLICT (pack_id) DO NOTHING`, packID, currentVersion, checksum); err != nil {
		return fmt.Errorf("jobs: write tombstone: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.cancelPack(packID)
	return nil
}

// Run starts the worker and returns when ctx is cancelled, so a caller must keep
// it running for the process lifetime. Startup recovery runs first.
func (r *Runner) Run(ctx context.Context) error {
	if err := r.Recover(ctx); err != nil {
		return err
	}
	for {
		job, err := r.claim(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if job != nil {
			r.execute(ctx, *job)
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-r.wake:
		}
	}
}

func (r *Runner) signal() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// claim promotes the oldest queued job to running in one short transaction.
func (r *Runner) claim(ctx context.Context) (*Job, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(ctx, `SELECT id FROM processing_jobs WHERE status = 'queued' ORDER BY rowid LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jobs: claim: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE processing_jobs SET status = 'running',
		started_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		error_code = '', error_message = '', finished_at = NULL
		WHERE id = ? AND status = 'queued'`, id)
	if err != nil {
		return nil, fmt.Errorf("jobs: claim: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return nil, fmt.Errorf("jobs: claim: %w", err)
	} else if affected == 0 {
		return nil, nil
	}
	job, err := scanJob(tx.QueryRowContext(ctx, jobSelect+` WHERE id = ?`, id))
	if err != nil {
		return nil, fmt.Errorf("jobs: claim: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &job, nil
}

// execute runs one claimed job: processor child, bounded validation, staged
// files, renames and finally the database publication.
func (r *Runner) execute(ctx context.Context, job Job) {
	r.childMu.Lock()
	defer r.childMu.Unlock()

	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	r.mu.Lock()
	r.activePack, r.cancelActive = job.PackID, cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.activePack, r.cancelActive = "", nil
		r.mu.Unlock()
	}()

	// Register first: a revoke either already persisted an interruption or can
	// now cancel this execution, including between this check and child startup.
	var runnable bool
	if err := r.db.QueryRowContext(runCtx, `SELECT EXISTS (
		SELECT 1 FROM processing_jobs j JOIN packs p ON p.id = j.pack_id
		WHERE j.id = ? AND j.status = 'running' AND p.status != 'revoked'
	)`, job.ID).Scan(&runnable); err != nil {
		switch {
		case errors.Is(runCtx.Err(), context.DeadlineExceeded):
			r.fail(ctx, job, CodeTimeout, fmt.Sprintf("processing did not finish within %s", r.timeout))
		case ctx.Err() != nil:
			r.finish(ctx, job, StatusInterrupted, CodeShutdown, "the server stopped while this job was running")
		default:
			r.fail(ctx, job, CodeProcessorFailed, err.Error())
		}
		return
	}
	if !runnable {
		return
	}

	request, err := r.prepare(job)
	if err != nil {
		r.fail(ctx, job, CodeProcessorFailed, err.Error())
		return
	}
	result, err := r.processor.Process(runCtx, request)
	if err != nil {
		switch {
		case errors.Is(runCtx.Err(), context.DeadlineExceeded):
			r.fail(ctx, job, CodeTimeout, fmt.Sprintf("processing did not finish within %s", r.timeout))
		case ctx.Err() != nil:
			r.finish(ctx, job, StatusInterrupted, CodeShutdown, "the server stopped while this job was running")
		default:
			r.fail(ctx, job, CodeProcessorFailed, err.Error())
		}
		return
	}
	manifest, err := jobManifest(request, result)
	if err != nil {
		r.fail(ctx, job, CodeResultInvalid, err.Error())
		return
	}
	staged, err := r.files.StageBundle(ctx, job.ID, manifest)
	if err != nil {
		r.fail(ctx, job, CodeStageFailed, err.Error())
		return
	}
	committed, err := r.files.CommitFiles(ctx, staged)
	if err != nil {
		r.fail(ctx, job, CodeCommitFailed, err.Error())
		return
	}
	err = packs.CommitProcessed(ctx, r.db, packs.ProcessedVersion{
		JobID:       job.ID,
		PackID:      job.PackID,
		Version:     job.Version,
		Manifest:    manifest,
		Files:       committed,
		Diagnostics: result.Diagnostics,
	})
	if err != nil {
		// A revoke or an earlier terminal state wins: the conditional update
		// inside CommitProcessed leaves the job and the pointer untouched.
		r.fail(ctx, job, CodePublishFailed, err.Error())
	}
}

// prepare resolves the server-owned paths and gives the child an empty output
// directory, so a stale artifact can never be interpreted as a result.
func (r *Runner) prepare(job Job) (JobRequest, error) {
	request := job.Request
	if !isUUID(request.Source.ID) {
		return JobRequest{}, fmt.Errorf("%w: request has no source", ErrInvalid)
	}
	sheetPath, err := r.files.SourcePath(request.Source.ID)
	if err != nil {
		return JobRequest{}, fmt.Errorf("resolve source sheet: %w", err)
	}
	workDir, err := r.files.WorkDir(job.ID)
	if err != nil {
		return JobRequest{}, err
	}
	outputDir := filepath.Join(workDir, "output")
	if err := os.RemoveAll(outputDir); err != nil {
		return JobRequest{}, fmt.Errorf("clear job output: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return JobRequest{}, fmt.Errorf("create job output: %w", err)
	}
	request.SheetPath = sheetPath
	request.WorkDir = workDir
	return request, nil
}

// jobManifest validates the processor result shape and freezes it into the
// immutable version manifest. Exact bytes and dimensions are re-verified from
// disk by assets.StageBundle.
func jobManifest(request JobRequest, result JobResult) (assets.PackManifest, error) {
	want := request.Rows * request.Columns
	if want < 1 || want > assets.MaxStickers {
		return assets.PackManifest{}, fmt.Errorf("%w: design has %d cells", errInvalidResult, want)
	}
	if len(result.Stickers) != want {
		return assets.PackManifest{}, fmt.Errorf("%w: %d stickers for %d cells", errInvalidResult, len(result.Stickers), want)
	}
	manifest := assets.PackManifest{
		PackID:      request.PackID,
		Slug:        request.Catalog.Slug,
		Name:        request.Catalog.Name,
		Author:      request.Catalog.Author,
		Description: request.Catalog.Description,
		Tags:        append([]string(nil), request.Catalog.Tags...),
		Version:     request.Version,
		Stickers:    make([]assets.ManifestSticker, want),
	}
	for i, sticker := range result.Stickers {
		switch {
		case sticker.File != fmt.Sprintf("%03d.webp", i):
			return assets.PackManifest{}, fmt.Errorf("%w: member %q is not the fixed name for index %d", errInvalidResult, sticker.File, i)
		case sticker.ThumbFile != fmt.Sprintf("thumb_%03d.webp", i):
			return assets.PackManifest{}, fmt.Errorf("%w: thumb %q is not the fixed name for index %d", errInvalidResult, sticker.ThumbFile, i)
		case sticker.Width != assets.StickerDimension || sticker.Height != assets.StickerDimension:
			return assets.PackManifest{}, fmt.Errorf("%w: %s is %dx%d, want %dx%d", errInvalidResult,
				sticker.File, sticker.Width, sticker.Height, assets.StickerDimension, assets.StickerDimension)
		}
		manifest.Stickers[i] = assets.ManifestSticker{
			File:      sticker.File,
			ThumbFile: sticker.ThumbFile,
			Width:     sticker.Width,
			Height:    sticker.Height,
		}
	}
	return manifest, nil
}

func (r *Runner) fail(ctx context.Context, job Job, code, message string) {
	r.finish(ctx, job, StatusFailed, code, message)
}

// finish writes a terminal state only while the job is still running, so it
// never overwrites an interrupted or succeeded job.
func (r *Runner) finish(ctx context.Context, job Job, status Status, code, message string) {
	if len(message) > maxMessageChars {
		message = message[:maxMessageChars]
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, _ = r.db.ExecContext(writeCtx, `UPDATE processing_jobs SET status = ?, error_code = ?, error_message = ?,
		finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND status = 'running'`, string(status), code, message, job.ID)
}

func (r *Runner) cancelPack(packID string) {
	r.mu.Lock()
	cancel := context.CancelFunc(nil)
	if r.activePack == packID {
		cancel = r.cancelActive
	}
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// enqueueSnapshot is the frozen pack/design state read inside the enqueue
// transaction.
type enqueueSnapshot struct {
	status         string
	designID       sql.NullString
	sourceID       sql.NullString
	sourceFilename string
	sourceChecksum string
	sourceWidth    int
	sourceHeight   int
	sourceByteSize int64
	catalog        CatalogRef
	design         studio.Snapshot
	rows           int
	columns        int
	outlineWidth   int
	outlineColor   string
}

func loadEnqueueSnapshot(ctx context.Context, tx *sql.Tx, packID string) (enqueueSnapshot, error) {
	var (
		snapshot       enqueueSnapshot
		rows           sql.NullInt64
		columns        sql.NullInt64
		outlineWidth   sql.NullInt64
		outlineColor   sql.NullString
		snapshotJSON   sql.NullString
		sourceWidth    sql.NullInt64
		sourceHeight   sql.NullInt64
		sourceByteSize sql.NullInt64
		designID       sql.NullString
		sourceID       sql.NullString
	)
	err := tx.QueryRowContext(ctx, `SELECT p.status, p.design_id, p.source_id, p.source_filename,
			p.source_checksum_sha256, p.source_width, p.source_height, p.source_byte_size,
			p.slug, p.name, p.author, p.description,
			d.rows, d.columns, d.outline_width, d.outline_color, d.snapshot_json
		FROM packs p LEFT JOIN designs d ON d.id = p.design_id
		WHERE p.id = ?`, packID).Scan(
		&snapshot.status, &designID, &sourceID, &snapshot.sourceFilename,
		&snapshot.sourceChecksum, &sourceWidth, &sourceHeight, &sourceByteSize,
		&snapshot.catalog.Slug, &snapshot.catalog.Name, &snapshot.catalog.Author, &snapshot.catalog.Description,
		&rows, &columns, &outlineWidth, &outlineColor, &snapshotJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return enqueueSnapshot{}, ErrNotFound
	}
	if err != nil {
		return enqueueSnapshot{}, fmt.Errorf("jobs: read pack: %w", err)
	}
	snapshot.designID = designID
	snapshot.sourceID = sourceID
	snapshot.sourceWidth = int(sourceWidth.Int64)
	snapshot.sourceHeight = int(sourceHeight.Int64)
	snapshot.sourceByteSize = sourceByteSize.Int64
	snapshot.rows = int(rows.Int64)
	snapshot.columns = int(columns.Int64)
	snapshot.outlineWidth = int(outlineWidth.Int64)
	snapshot.outlineColor = outlineColor.String
	if snapshotJSON.Valid {
		if err := json.Unmarshal([]byte(snapshotJSON.String), &snapshot.design); err != nil {
			return enqueueSnapshot{}, fmt.Errorf("%w: design snapshot: %v", ErrInvalid, err)
		}
	}
	tags, err := loadTags(ctx, tx, packID)
	if err != nil {
		return enqueueSnapshot{}, err
	}
	snapshot.catalog.Tags = tags
	return snapshot, nil
}

func loadTags(ctx context.Context, tx *sql.Tx, packID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT tag FROM pack_tags WHERE pack_id = ? ORDER BY tag`, packID)
	if err != nil {
		return nil, fmt.Errorf("jobs: read tags: %w", err)
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("jobs: scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: read tags: %w", err)
	}
	return tags, nil
}

func (s enqueueSnapshot) sourceRef(sourceID string) (SourceRef, error) {
	if !s.sourceID.Valid || s.sourceFilename == "" {
		return SourceRef{}, ErrNoSource
	}
	if sourceID != "" && sourceID != s.sourceID.String {
		return SourceRef{}, fmt.Errorf("%w: source does not belong to the pack", ErrInvalid)
	}
	return SourceRef{
		ID:             s.sourceID.String,
		Filename:       s.sourceFilename,
		ChecksumSHA256: s.sourceChecksum,
		Width:          s.sourceWidth,
		Height:         s.sourceHeight,
		ByteSize:       s.sourceByteSize,
	}, nil
}

func (s enqueueSnapshot) designRef(designID string) (string, error) {
	switch {
	case !s.designID.Valid || s.designID.String == "":
		return "", fmt.Errorf("%w: pack has no design", ErrInvalid)
	case designID != "" && designID != s.designID.String:
		return "", fmt.Errorf("%w: design does not belong to the pack", ErrInvalid)
	case s.rows < 1 || s.rows > 8 || s.columns < 1 || s.columns > 8:
		return "", fmt.Errorf("%w: design rows %d columns %d", ErrInvalid, s.rows, s.columns)
	}
	if err := validateSnapshot(s.design, s.rows, s.columns); err != nil {
		return "", err
	}
	return s.designID.String, nil
}

// validateSnapshot pins the row-major 1-based cell order the processor output
// and sticker sort_order depend on.
func validateSnapshot(snapshot studio.Snapshot, rows, columns int) error {
	if snapshot.SchemaVersion != 1 {
		return fmt.Errorf("%w: design snapshot schema %d", ErrInvalid, snapshot.SchemaVersion)
	}
	if len(snapshot.Cells) != rows*columns {
		return fmt.Errorf("%w: design snapshot has %d cells for %dx%d", ErrInvalid, len(snapshot.Cells), rows, columns)
	}
	for i, cell := range snapshot.Cells {
		if cell.Row != i/columns+1 || cell.Column != i%columns+1 {
			return fmt.Errorf("%w: design snapshot cell %d is not row-major", ErrInvalid, i)
		}
	}
	return nil
}

func scanJob(row *sql.Row) (Job, error) {
	var (
		job         Job
		status      string
		requestJSON string
		resultJSON  sql.NullString
		createdAt   string
		startedAt   sql.NullString
		finishedAt  sql.NullString
	)
	if err := row.Scan(&job.ID, &job.PackID, &job.Version, &status, &requestJSON, &resultJSON,
		&job.ErrorCode, &job.ErrorMessage, &job.Diagnostics, &createdAt, &startedAt, &finishedAt); err != nil {
		return Job{}, err
	}
	job.Status = Status(status)
	if err := json.Unmarshal([]byte(requestJSON), &job.Request); err != nil {
		return Job{}, fmt.Errorf("%w: request json: %v", ErrInvalid, err)
	}
	if resultJSON.Valid {
		job.ResultJSON = json.RawMessage(resultJSON.String)
	}
	job.CreatedAt = parseTimestamp(createdAt)
	job.StartedAt = parseTimestamp(startedAt.String)
	job.FinishedAt = parseTimestamp(finishedAt.String)
	return job, nil
}

func parseTimestamp(value string) time.Time {
	parsed, err := time.Parse(timestampLayout, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func hasActiveJob(ctx context.Context, tx *sql.Tx, packID string) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM processing_jobs
		WHERE pack_id = ? AND status IN ('queued', 'running') LIMIT 1`, packID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("jobs: check active job: %w", err)
	}
	return true, nil
}

func isUUID(value string) bool {
	return uuid.Validate(value) == nil
}

// isActiveJobConflict maps the one-active-job and version unique indexes to the
// single concurrency error the admin API reports.
func isActiveJobConflict(err error) bool {
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") && strings.Contains(message, "processing_jobs")
}
