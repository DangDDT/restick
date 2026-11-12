package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/eykh/restick/api_svc/internal/studio"
	"github.com/google/uuid"
)

// Real lossless WebP at the fixed output sizes of spec §7.2: 512x512 stickers
// and 128x128 thumbnails. assets.StageBundle verifies the decoded dimensions of
// every staged member.
const (
	stickerWebP = "UklGRi4AAABXRUJQVlA4TCIAAAAv/8F/AAdQlHIXpf8BgUCyv/cMRfQ/4z//+c9//vOf//wf"
	thumbWebP   = "UklGRhgAAABXRUJQVlA4TAsAAAAvf8AfAEVRShv9DwA="
)

type harness struct {
	t       *testing.T
	db      *sql.DB
	store   *assets.Store
	catalog *packs.Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := assets.NewStore(dir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	return &harness{t: t, db: db, store: store, catalog: packs.NewService(db)}
}

// newPack creates a 2x2 design, a draft pack that uses it and a stored source
// sheet, so enqueue has a full snapshot to freeze.
func (h *harness) newPack(slug string) (packID, designID, sourceID string) {
	h.t.Helper()
	ctx := context.Background()
	designID = uuid.NewString()
	snapshot := studio.Snapshot{
		SchemaVersion: 1,
		Rows:          2,
		Columns:       2,
		OutlineWidth:  4,
		OutlineColor:  "#FFFFFF",
		CompiledAt:    "2026-11-10T00:00:00Z",
	}
	for row := 1; row <= 2; row++ {
		for column := 1; column <= 2; column++ {
			snapshot.Cells = append(snapshot.Cells, studio.SnapshotCell{Row: row, Column: column})
		}
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		h.t.Fatalf("marshal snapshot: %v", err)
	}
	if _, err := h.db.Exec(`INSERT INTO designs
		(id, name, rows, columns, outline_width, outline_color, draft_json, snapshot_json)
		VALUES (?, ?, 2, 2, 4, '#FFFFFF', '{"schema_version":1}', ?)`,
		designID, "Design "+slug, string(snapshotJSON)); err != nil {
		h.t.Fatalf("insert design: %v", err)
	}
	sheet, err := studio.NewService(h.db, h.store).SaveSheet(ctx, designID,
		bytes.NewReader(encodePNG(h.t, 64, 64)), "sheet.png", "gemini", "")
	if err != nil {
		h.t.Fatalf("save sheet: %v", err)
	}
	pack, err := h.catalog.Create(ctx, packs.PackDraft{
		Slug:     slug,
		Name:     "Pack " + slug,
		Author:   "Tester",
		DesignID: designID,
		SheetID:  sheet.ID,
		Tags:     []string{"demo"},
	})
	if err != nil {
		h.t.Fatalf("create pack: %v", err)
	}
	return pack.ID, designID, sheet.ID
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: 200, G: 40, B: 40, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func webpBytes(t *testing.T, encoded string) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode webp fixture: %v", err)
	}
	return data
}

// fakeProcessor records overlap, so a second concurrent child is detectable.
type fakeProcessor struct {
	process func(ctx context.Context, request JobRequest) (JobResult, error)

	mu        sync.Mutex
	active    int
	maxActive int
	calls     []JobRequest
}

func (f *fakeProcessor) Process(ctx context.Context, request JobRequest) (JobResult, error) {
	f.mu.Lock()
	f.active++
	if f.active > f.maxActive {
		f.maxActive = f.active
	}
	f.calls = append(f.calls, request)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		f.mu.Unlock()
	}()
	return f.process(ctx, request)
}

func (f *fakeProcessor) overlap() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxActive
}

// writeProcessedOutput writes the fixed flat work-output names into the job
// directory, exactly like the Python machine job does.
func writeProcessedOutput(t *testing.T, request JobRequest, count int) (JobResult, error) {
	output := filepath.Join(request.WorkDir, "output")
	result := JobResult{Diagnostics: `{"engine":"fake"}`}
	for i := range count {
		file := fmt.Sprintf("%03d.webp", i)
		thumb := "thumb_" + file
		files := map[string]string{file: stickerWebP, thumb: thumbWebP}
		for name, encoded := range files {
			if err := os.WriteFile(filepath.Join(output, name), webpBytes(t, encoded), 0o644); err != nil {
				t.Errorf("write work output: %v", err)
				return JobResult{}, err
			}
		}
		result.Stickers = append(result.Stickers, ResultSticker{
			File: file, ThumbFile: thumb,
			Width: assets.StickerDimension, Height: assets.StickerDimension,
		})
	}
	return result, nil
}

func succeed(t *testing.T) func(context.Context, JobRequest) (JobResult, error) {
	return func(_ context.Context, request JobRequest) (JobResult, error) {
		return writeProcessedOutput(t, request, request.Rows*request.Columns)
	}
}

// runWorker starts Run in its own goroutine and stops it with the test.
func runWorker(t *testing.T, runner *Runner) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("worker stopped with %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("worker did not stop")
		}
	})
}

func waitJob(t *testing.T, runner *Runner, id string, want Status) Job {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		job, err := runner.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("get job %s: %v", id, err)
		}
		if job.Status == want {
			return job
		}
		if terminal(job.Status) {
			t.Fatalf("job %s reached %s, want %s: %s", id, job.Status, want, job.ErrorMessage)
		}
		if time.Now().After(deadline) {
			t.Fatalf("job %s is still %s after waiting for %s", id, job.Status, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func terminal(status Status) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusInterrupted:
		return true
	default:
		return false
	}
}

func TestEnqueueUsesSheetSource(t *testing.T) {
	h := newHarness(t)
	packID, designID, sheetID := h.newPack("sheet-src")
	var checksum string
	if err := h.db.QueryRow(`SELECT checksum_sha256 FROM design_sheets WHERE id = ?`, sheetID).Scan(&checksum); err != nil {
		t.Fatalf("read sheet: %v", err)
	}
	job, err := NewRunner(h.db, h.store, nil, time.Second).Enqueue(context.Background(), packID, sheetID, designID)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	var raw string
	if err := h.db.QueryRow(`SELECT request_json FROM processing_jobs WHERE id = ?`, job.ID).Scan(&raw); err != nil {
		t.Fatalf("read request_json: %v", err)
	}
	var request JobRequest
	if err := json.Unmarshal([]byte(raw), &request); err != nil {
		t.Fatalf("decode request_json: %v", err)
	}
	if request.Source.ID != sheetID || request.Source.ChecksumSHA256 != checksum {
		t.Fatalf("source = %+v, want id %s checksum %s", request.Source, sheetID, checksum)
	}
	if _, err := NewRunner(h.db, h.store, nil, time.Second).Enqueue(context.Background(), packID, uuid.NewString(), designID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("foreign source err = %v, want ErrInvalid", err)
	}
}

// TestEnqueueKeepsExtendedDesignSnapshot proves the schema-1 snapshot handoff
// still freezes the optional label and selected reference appended by Task 2.
func TestEnqueueKeepsExtendedDesignSnapshot(t *testing.T) {
	h := newHarness(t)
	packID, designID, sourceID := h.newPack("extended-snapshot")

	snapshot := map[string]any{
		"schema_version": 1,
		"character": map[string]any{
			"id": "char-1", "name": "Sen Con", "description": "a lotus sprite",
			"identity_notes": "pink bud hat", "reference_asset_id": "asset-1",
		},
		"selections": []any{}, "free_text": "three Vietnamese dishes",
		"rows": 1, "columns": 2, "outline_width": 4, "outline_color": "#FFFFFF",
		"cells": []any{
			map[string]any{"row": 1, "column": 1, "action": "steaming", "expression": "happy",
				"free_text": "a beef noodle bowl", "fragments": []any{}, "label": "Bún bò Huế"},
			map[string]any{"row": 1, "column": 2, "action": "sitting", "expression": "calm",
				"free_text": "a pho plate", "fragments": []any{}},
		},
		"compiled_at": "2026-11-10T00:00:00Z",
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if _, err := h.db.Exec(`UPDATE designs SET rows = 1, columns = 2, snapshot_json = ? WHERE id = ?`,
		string(raw), designID); err != nil {
		t.Fatalf("store extended snapshot: %v", err)
	}

	job, err := NewRunner(h.db, h.store, nil, time.Second).Enqueue(context.Background(), packID, sourceID, designID)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	var stored string
	if err := h.db.QueryRow(`SELECT request_json FROM processing_jobs WHERE id = ?`, job.ID).Scan(&stored); err != nil {
		t.Fatalf("read request_json: %v", err)
	}
	var request map[string]any
	if err := json.Unmarshal([]byte(stored), &request); err != nil {
		t.Fatalf("decode request_json: %v", err)
	}
	if request["schema_version"] != float64(1) {
		t.Fatalf("request schema_version = %v, want 1", request["schema_version"])
	}
	design, ok := request["design"].(map[string]any)
	if !ok {
		t.Fatalf("request has no design snapshot: %v", request)
	}
	if design["schema_version"] != float64(1) {
		t.Fatalf("design schema_version = %v, want 1", design["schema_version"])
	}
	if design["free_text"] != "three Vietnamese dishes" {
		t.Fatalf("design free_text = %v", design["free_text"])
	}
	character, ok := design["character"].(map[string]any)
	if !ok || character["reference_asset_id"] != "asset-1" {
		t.Fatalf("design character = %v, want the frozen reference", design["character"])
	}
	cells, ok := design["cells"].([]any)
	if !ok || len(cells) != 2 {
		t.Fatalf("design cells = %v, want two", design["cells"])
	}
	first, _ := cells[0].(map[string]any)
	second, _ := cells[1].(map[string]any)
	if first["row"] != float64(1) || first["column"] != float64(1) || first["label"] != "Bún bò Huế" {
		t.Fatalf("first cell = %v, want row-major label", first)
	}
	if second["row"] != float64(1) || second["column"] != float64(2) {
		t.Fatalf("second cell = %v, want the next row-major cell", second)
	}
	if label, ok := second["label"]; ok && label != "" {
		t.Fatalf("second cell label = %v, want none", label)
	}

	// The extended document still decodes as a schema-1 job request.
	var decoded JobRequest
	if err := json.Unmarshal([]byte(stored), &decoded); err != nil {
		t.Fatalf("decode JobRequest: %v", err)
	}
	if decoded.SchemaVersion != 1 || decoded.Design.SchemaVersion != 1 ||
		len(decoded.Design.Cells) != 2 || decoded.Rows != 1 || decoded.Columns != 2 {
		t.Fatalf("decoded request = schema %d design schema %d cells %d grid %dx%d",
			decoded.SchemaVersion, decoded.Design.SchemaVersion, len(decoded.Design.Cells),
			decoded.Rows, decoded.Columns)
	}
}

func TestEnqueueWithoutSheetReturnsErrNoSource(t *testing.T) {
	h := newHarness(t)
	packID, designID, _ := h.newPack("no-sheet")
	if _, err := h.db.Exec(`UPDATE packs SET sheet_id = NULL WHERE id = ?`, packID); err != nil {
		t.Fatalf("clear sheet: %v", err)
	}
	_, err := NewRunner(h.db, h.store, nil, time.Second).Enqueue(context.Background(), packID, "", designID)
	if !errors.Is(err, ErrNoSource) {
		t.Fatalf("err = %v, want ErrNoSource", err)
	}
}

func TestConcurrentEnqueueOneActiveJob(t *testing.T) {
	h := newHarness(t)
	packID, designID, sourceID := h.newPack("demo-one")
	runner := NewRunner(h.db, h.store, nil, time.Second)

	const workers = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, workers)
	versions := make([]int, workers)
	for i := range workers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			job, err := runner.Enqueue(context.Background(), packID, sourceID, designID)
			errs[index] = err
			versions[index] = job.Version
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range errs {
		switch {
		case err == nil:
			succeeded++
			if versions[i] != 1 {
				t.Errorf("accepted job reserved version %d, want 1", versions[i])
			}
		case errors.Is(err, ErrJobActive):
		default:
			t.Errorf("enqueue %d: %v", i, err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("%d concurrent enqueues succeeded, want exactly 1", succeeded)
	}
	var queued int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM processing_jobs WHERE pack_id = ? AND status = 'queued'`, packID).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("%d queued jobs for the pack, want 1", queued)
	}
}

func TestWorkerNeverRunsTwoChildren(t *testing.T) {
	h := newHarness(t)
	fake := &fakeProcessor{process: func(_ context.Context, request JobRequest) (JobResult, error) {
		time.Sleep(20 * time.Millisecond)
		return writeProcessedOutput(t, request, request.Rows*request.Columns)
	}}
	runner := NewRunner(h.db, h.store, fake, 10*time.Second)
	ids := make([]string, 0, 3)
	for i := range 3 {
		packID, designID, sourceID := h.newPack(fmt.Sprintf("demo-%d", i))
		job, err := runner.Enqueue(context.Background(), packID, sourceID, designID)
		if err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
		ids = append(ids, job.ID)
	}
	runWorker(t, runner)
	for _, id := range ids {
		job := waitJob(t, runner, id, StatusSucceeded)
		pack, err := h.catalog.Get(context.Background(), job.PackID)
		if err != nil {
			t.Fatalf("get pack: %v", err)
		}
		if pack.CurrentVersion != job.Version || pack.Status != packs.StatusReady {
			t.Errorf("pack %s is %s v%d, want ready v%d", pack.ID, pack.Status, pack.CurrentVersion, job.Version)
		}
	}
	if overlap := fake.overlap(); overlap != 1 {
		t.Fatalf("%d processor children overlapped, want exactly 1 at a time", overlap)
	}
}

func TestStartupInterruptsRunningAndResumesQueued(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	stalePack, staleDesign, staleSource := h.newPack("stale-pack")
	staleJob, err := NewRunner(h.db, h.store, nil, time.Second).Enqueue(ctx, stalePack, staleSource, staleDesign)
	if err != nil {
		t.Fatalf("enqueue stale job: %v", err)
	}
	if _, err := h.db.Exec(`UPDATE processing_jobs SET status = 'running',
		started_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`, staleJob.ID); err != nil {
		t.Fatalf("simulate previous run: %v", err)
	}

	queuedPack, queuedDesign, queuedSource := h.newPack("queued-pack")
	fake := &fakeProcessor{process: succeed(t)}
	runner := NewRunner(h.db, h.store, fake, 10*time.Second)
	resumed, err := runner.Enqueue(ctx, queuedPack, queuedSource, queuedDesign)
	if err != nil {
		t.Fatalf("enqueue queued job: %v", err)
	}
	runWorker(t, runner)
	waitJob(t, runner, resumed.ID, StatusSucceeded)

	stale, err := runner.Get(ctx, staleJob.ID)
	if err != nil {
		t.Fatalf("get stale job: %v", err)
	}
	if stale.Status != StatusInterrupted || stale.ErrorCode != CodeStartupRecovery || stale.FinishedAt.IsZero() {
		t.Fatalf("stale job is %s/%s finished %v, want interrupted/%s", stale.Status, stale.ErrorCode, stale.FinishedAt, CodeStartupRecovery)
	}
	pack, err := h.catalog.Get(ctx, stalePack)
	if err != nil {
		t.Fatalf("get stale pack: %v", err)
	}
	if pack.Status != packs.StatusDraft || pack.CurrentVersion != 0 {
		t.Fatalf("stale pack is %s v%d, want draft v0", pack.Status, pack.CurrentVersion)
	}
}

func TestTimeoutKeepsOldReadyVersion(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID, designID, sourceID := h.newPack("timeout-pack")
	var calls atomic.Int32
	fake := &fakeProcessor{process: func(ctx context.Context, request JobRequest) (JobResult, error) {
		if calls.Add(1) == 1 {
			return writeProcessedOutput(t, request, request.Rows*request.Columns)
		}
		<-ctx.Done()
		return JobResult{}, ctx.Err()
	}}
	runner := NewRunner(h.db, h.store, fake, 300*time.Millisecond)
	first, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}
	runWorker(t, runner)
	waitJob(t, runner, first.ID, StatusSucceeded)

	second, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue second job: %v", err)
	}
	failed := waitJob(t, runner, second.ID, StatusFailed)
	if failed.ErrorCode != CodeTimeout {
		t.Fatalf("timeout job error code = %q, want %q", failed.ErrorCode, CodeTimeout)
	}
	pack, err := h.catalog.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != packs.StatusReady || pack.CurrentVersion != first.Version {
		t.Fatalf("pack is %s v%d, want ready v%d after a timeout", pack.Status, pack.CurrentVersion, first.Version)
	}
	if _, err := h.catalog.GetVersion(ctx, packID, first.Version); err != nil {
		t.Fatalf("old version is not downloadable: %v", err)
	}
	if _, err := h.catalog.GetVersion(ctx, packID, second.Version); !errors.Is(err, packs.ErrVersionNotFound) {
		t.Fatalf("timed-out version error = %v, want ErrVersionNotFound", err)
	}
}

func TestCrashWindowDoesNotExposeVersion(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID, designID, sourceID := h.newPack("crash-pack")
	runner := NewRunner(h.db, h.store, nil, time.Second)
	job, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// A worker renamed the complete version into place and then died before the
	// database commit.
	workDir, err := h.store.WorkDir(job.ID)
	if err != nil {
		t.Fatalf("work dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "output"), 0o755); err != nil {
		t.Fatalf("create output: %v", err)
	}
	manifest := assets.PackManifest{
		PackID: packID, Slug: "crash-pack", Name: "Pack crash-pack", Author: "Tester",
		Tags: []string{"demo"}, Version: job.Version,
	}
	for i := range 4 {
		file := fmt.Sprintf("%03d.webp", i)
		thumb := "thumb_" + file
		files := map[string]string{file: stickerWebP, thumb: thumbWebP}
		for name, encoded := range files {
			if err := os.WriteFile(filepath.Join(workDir, "output", name), webpBytes(t, encoded), 0o644); err != nil {
				t.Fatalf("write output: %v", err)
			}
		}
		manifest.Stickers = append(manifest.Stickers, assets.ManifestSticker{
			File: file, ThumbFile: thumb,
			Width: assets.StickerDimension, Height: assets.StickerDimension,
		})
	}
	staged, err := h.store.StageBundle(ctx, job.ID, manifest)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	committed, err := h.store.CommitFiles(ctx, staged)
	if err != nil {
		t.Fatalf("commit files: %v", err)
	}
	if _, err := h.db.Exec(`UPDATE processing_jobs SET status = 'running',
		started_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`, job.ID); err != nil {
		t.Fatalf("simulate crash: %v", err)
	}

	if err := runner.Recover(ctx); err != nil {
		t.Fatalf("recover: %v", err)
	}
	interrupted, err := runner.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if interrupted.Status != StatusInterrupted {
		t.Fatalf("crashed job is %s, want interrupted", interrupted.Status)
	}
	pack, err := h.catalog.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != packs.StatusDraft || pack.CurrentVersion != 0 {
		t.Fatalf("crash window published the pack as %s v%d", pack.Status, pack.CurrentVersion)
	}
	if _, err := h.catalog.GetVersion(ctx, packID, job.Version); !errors.Is(err, packs.ErrVersionNotFound) {
		t.Fatalf("crash-window version error = %v, want ErrVersionNotFound", err)
	}
	if _, err := h.store.OpenCommitted(committed.OutputDir + "/pack.json"); !errors.Is(err, assets.ErrNotFound) {
		t.Errorf("unreferenced version files were kept: %v", err)
	}
	if _, err := h.store.OpenCommitted(committed.BundlePath); !errors.Is(err, assets.ErrNotFound) {
		t.Errorf("unreferenced bundle was kept: %v", err)
	}
}

func TestRevokeWinsLateCompletion(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID, designID, sourceID := h.newPack("revoke-pack")
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	fake := &fakeProcessor{process: func(ctx context.Context, request JobRequest) (JobResult, error) {
		if calls.Add(1) == 1 {
			return writeProcessedOutput(t, request, request.Rows*request.Columns)
		}
		close(started)
		<-release
		return writeProcessedOutput(t, request, request.Rows*request.Columns)
	}}
	runner := NewRunner(h.db, h.store, fake, 30*time.Second)
	first, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}
	runWorker(t, runner)
	waitJob(t, runner, first.ID, StatusSucceeded)

	second, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue second job: %v", err)
	}
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("second child never started")
	}
	if err := runner.Revoke(ctx, packID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	revoked, err := runner.Get(ctx, second.ID)
	if err != nil {
		t.Fatalf("get revoked job: %v", err)
	}
	if revoked.Status != StatusInterrupted || revoked.ErrorCode != CodePackRevoked {
		t.Fatalf("running job is %s/%s, want interrupted/%s", revoked.Status, revoked.ErrorCode, CodePackRevoked)
	}

	// The child returns a perfectly valid result after the revoke, so the
	// completion reaches the file rename and the database guard.
	close(release)
	runner.childMu.Lock()
	runner.childMu.Unlock()

	late, err := runner.Get(ctx, second.ID)
	if err != nil {
		t.Fatalf("get late job: %v", err)
	}
	if late.Status != StatusInterrupted {
		t.Fatalf("late completion resurrected the job as %s", late.Status)
	}
	pack, err := h.catalog.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != packs.StatusRevoked || pack.CurrentVersion != first.Version {
		t.Fatalf("pack is %s v%d, want revoked v%d", pack.Status, pack.CurrentVersion, first.Version)
	}
	if _, err := h.catalog.GetVersion(ctx, packID, second.Version); !errors.Is(err, packs.ErrVersionNotFound) {
		t.Fatalf("late version error = %v, want ErrVersionNotFound", err)
	}
	if count, version, checksum := tombstone(t, h.db, packID); count != 1 || version != first.Version || checksum == "" {
		t.Fatalf("tombstone = %d rows v%d checksum %q", count, version, checksum)
	}
	if err := runner.Revoke(ctx, packID); err != nil {
		t.Fatalf("second revoke: %v", err)
	}
	if count, _, _ := tombstone(t, h.db, packID); count != 1 {
		t.Fatalf("%d tombstones after a repeated revoke, want 1", count)
	}
}

func TestRevokeBeforeExecutionDoesNotStartProcessor(t *testing.T) {
	h := newHarness(t)
	var calls int
	fake := &fakeProcessor{process: func(ctx context.Context, request JobRequest) (JobResult, error) {
		calls++
		return writeProcessedOutput(t, request, request.Rows*request.Columns)
	}}
	runner := NewRunner(h.db, h.store, fake, time.Minute)
	job := claimReplacement(t, h, runner)
	if err := runner.Revoke(context.Background(), job.PackID); err != nil {
		t.Fatal(err)
	}
	runner.execute(context.Background(), job)
	if calls != 1 {
		t.Fatalf("processor ran %d times, want only the committed first version", calls)
	}
	interrupted := waitJob(t, runner, job.ID, StatusInterrupted)
	if interrupted.ErrorCode != CodePackRevoked {
		t.Fatalf("revoked job error = %q", interrupted.ErrorCode)
	}
}

func TestRevokeCancelsActiveProcessor(t *testing.T) {
	h := newHarness(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	var calls int
	fake := &fakeProcessor{process: func(ctx context.Context, request JobRequest) (JobResult, error) {
		calls++
		if calls == 1 {
			return writeProcessedOutput(t, request, request.Rows*request.Columns)
		}
		close(started)
		<-ctx.Done()
		close(cancelled)
		return JobResult{}, ctx.Err()
	}}
	runner := NewRunner(h.db, h.store, fake, time.Minute)
	job := claimReplacement(t, h, runner)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runner.execute(ctx, job)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("replacement processor never started")
	}
	if err := runner.Revoke(ctx, job.PackID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(10 * time.Second):
		t.Fatal("revoke did not cancel the active processor")
	}
	<-done
	interrupted := waitJob(t, runner, job.ID, StatusInterrupted)
	if interrupted.ErrorCode != CodePackRevoked {
		t.Fatalf("revoked job error = %q", interrupted.ErrorCode)
	}
	pack, err := h.catalog.Get(context.Background(), job.PackID)
	if err != nil || pack.Status != packs.StatusRevoked || pack.CurrentVersion != 1 {
		t.Fatalf("revoke changed committed version: %+v, %v", pack, err)
	}
}

// claimReplacement leaves a ready pack's second job between claim and execute.
func claimReplacement(t *testing.T, h *harness, runner *Runner) Job {
	t.Helper()
	ctx := context.Background()
	packID, designID, sourceID := h.newPack("claimed-replacement")
	if _, err := runner.Enqueue(ctx, packID, sourceID, designID); err != nil {
		t.Fatal(err)
	}
	first, err := runner.claim(ctx)
	if err != nil || first == nil {
		t.Fatalf("claim first job: %+v, %v", first, err)
	}
	runner.execute(ctx, *first)
	waitJob(t, runner, first.ID, StatusSucceeded)
	if _, err := runner.Enqueue(ctx, packID, sourceID, designID); err != nil {
		t.Fatal(err)
	}
	second, err := runner.claim(ctx)
	if err != nil || second == nil {
		t.Fatalf("claim replacement: %+v, %v", second, err)
	}
	return *second
}

func TestRevokeRejectsDraftPackWithActiveJob(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID, designID, sourceID := h.newPack("draft-pack")
	runner := NewRunner(h.db, h.store, nil, time.Second)
	job, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := runner.Revoke(ctx, packID); !errors.Is(err, ErrJobActive) {
		t.Fatalf("revoke of a draft with a queued job = %v, want ErrJobActive", err)
	}
	pack, err := h.catalog.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != packs.StatusDraft {
		t.Fatalf("pack status = %s, want draft", pack.Status)
	}
	current, err := runner.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if current.Status != StatusQueued {
		t.Fatalf("job status = %s, want queued", current.Status)
	}
	if count, _, _ := tombstone(t, h.db, packID); count != 0 {
		t.Fatalf("%d tombstones for a rejected revoke", count)
	}
}

func TestFailedVersionNumberIsNotReused(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID, designID, sourceID := h.newPack("retry-pack")
	var calls atomic.Int32
	fake := &fakeProcessor{process: func(ctx context.Context, request JobRequest) (JobResult, error) {
		if calls.Add(1) == 1 {
			return JobResult{}, errors.New("layout_invalid: missing candidate boxes")
		}
		return writeProcessedOutput(t, request, request.Rows*request.Columns)
	}}
	runner := NewRunner(h.db, h.store, fake, 10*time.Second)
	first, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}
	if first.Version != 1 {
		t.Fatalf("first version = %d, want 1", first.Version)
	}
	runWorker(t, runner)
	failed := waitJob(t, runner, first.ID, StatusFailed)
	if failed.ErrorCode != CodeProcessorFailed {
		t.Fatalf("failed job error code = %q, want %q", failed.ErrorCode, CodeProcessorFailed)
	}
	second, err := runner.Enqueue(ctx, packID, sourceID, designID)
	if err != nil {
		t.Fatalf("enqueue second job: %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("retry reserved version %d, want 2 (failed versions are never reused)", second.Version)
	}
	waitJob(t, runner, second.ID, StatusSucceeded)
	pack, err := h.catalog.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != packs.StatusReady || pack.CurrentVersion != 2 {
		t.Fatalf("pack is %s v%d, want ready v2", pack.Status, pack.CurrentVersion)
	}
}

func tombstone(t *testing.T, db *sql.DB, packID string) (count, version int, checksum string) {
	t.Helper()
	rows, err := db.Query(`SELECT version, checksum_sha256 FROM pack_tombstones WHERE pack_id = ?`, packID)
	if err != nil {
		t.Fatalf("read tombstones: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		count++
		if err := rows.Scan(&version, &checksum); err != nil {
			t.Fatalf("scan tombstone: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read tombstones: %v", err)
	}
	return count, version, checksum
}
