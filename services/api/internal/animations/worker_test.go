package animations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/google/uuid"
)

// animatedWebP is a minimal VP8X header with the animation bit: enough for
// webp.DecodeConfig, which is all the worker may call on animated files.
func animatedWebP(width, height int) []byte {
	w, h := width-1, height-1
	chunk := []byte{'V', 'P', '8', 'X', 10, 0, 0, 0,
		1 << 1, 0, 0, 0,
		byte(w), byte(w >> 8), byte(w >> 16), byte(h), byte(h >> 8), byte(h >> 16)}
	size := 4 + len(chunk)
	return append([]byte{'R', 'I', 'F', 'F', byte(size), byte(size >> 8), 0, 0, 'W', 'E', 'B', 'P'}, chunk...)
}

// fakeClip writes data as the child would and reports result; hook runs first.
type fakeClip struct {
	mu       sync.Mutex
	calls    []ClipRequest
	data     []byte
	result   func(r ClipRequest) ClipResult
	err      error
	hook     func(r ClipRequest)
	notifyCh chan struct{}
}

func (f *fakeClip) ProcessClip(ctx context.Context, r ClipRequest) (ClipResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, r)
	f.mu.Unlock()
	if f.notifyCh != nil {
		f.notifyCh <- struct{}{}
	}
	if f.hook != nil {
		f.hook(r)
	}
	if f.err != nil {
		return ClipResult{}, f.err
	}
	name := fmt.Sprintf("animated/%03d.webp", r.StickerIndex)
	if err := os.MkdirAll(filepath.Join(r.OutputDir, "animated"), 0o755); err != nil {
		return ClipResult{}, err
	}
	if err := os.WriteFile(filepath.Join(r.OutputDir, filepath.FromSlash(name)), f.data, 0o644); err != nil {
		return ClipResult{}, err
	}
	if f.result != nil {
		return f.result(r), nil
	}
	return ClipResult{Filename: name, Frames: 24, FPS: 12, DurationMS: 24 * 83, ByteSize: len(f.data),
		Quality: 75, Width: 512, Height: 512}, nil
}

func (f *fakeClip) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (h *harness) setDesign(packID string, width int, color string) {
	h.t.Helper()
	id := uuid.NewString()
	h.exec(`INSERT INTO designs (id, name, rows, columns, outline_width, outline_color, draft_json, snapshot_json)
		VALUES (?, 'D', 1, 2, ?, ?, '{}', '{"schema_version":1}')`, id, width, color)
	h.exec(`UPDATE packs SET design_id = ? WHERE id = ?`, id, packID)
}

// runWorker starts the worker and returns a stop function that waits for it.
func (h *harness) runWorker(p ClipProcessor, lock *sync.Mutex) func() {
	h.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.service.RunWorker(ctx, p, lock) }()
	stop := func() {
		cancel()
		if err := <-done; err != nil {
			h.t.Errorf("RunWorker: %v", err)
		}
	}
	h.t.Cleanup(func() {
		cancel()
	})
	return stop
}

func (h *harness) status(id string) (string, string) {
	h.t.Helper()
	var status, code string
	if err := h.db.QueryRow(`SELECT status, error_code FROM sticker_animations WHERE id = ?`, id).Scan(&status, &code); err != nil {
		h.t.Fatalf("read status: %v", err)
	}
	return status, code
}

func (h *harness) waitStatus(id string, want ...Status) string {
	h.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, _ := h.status(id)
		for _, w := range want {
			if status == string(w) {
				return status
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	status, code := h.status(id)
	h.t.Fatalf("animation %s is %s/%s, want %v", id, status, code, want)
	return ""
}

func TestWorkerMarksReadyAndStoresResult(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'), sum('b'))
	h.setDesign(packID, 6, "#112233")
	item := h.upload(packID, 1)
	data := animatedWebP(512, 512)
	fake := &fakeClip{data: data}
	stop := h.runWorker(fake, &sync.Mutex{})
	h.waitStatus(item.ID, StatusReady)
	stop()

	got, err := h.service.get(context.Background(), packID, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	digest := sha256.Sum256(data)
	if got.AnimatedPath != "animations/"+item.ID+".webp" || got.AnimatedChecksum != hex.EncodeToString(digest[:]) ||
		got.AnimatedByteSize != int64(len(data)) || got.Frames != 24 || got.FPS != 12 || got.DurationMS != 24*83 ||
		got.Quality != 75 || got.Width != 512 || got.Height != 512 || got.ErrorCode != "" || !got.UseAnimated {
		t.Fatalf("ready animation = %+v", got)
	}
	stored, err := os.ReadFile(filepath.Join(h.dir, "animations", item.ID+".webp"))
	if err != nil || string(stored) != string(data) {
		t.Fatalf("stored animated file: %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("processor calls = %d", len(fake.calls))
	}
	req := fake.calls[0]
	clipPath, _ := h.service.files.ClipPath(clipIDOf(t, h, item.ID))
	if req.AnimationID != item.ID || req.PackID != packID || req.StickerIndex != 1 || req.ClipPath != clipPath ||
		req.FPS != 12 || req.Quality != 75 || req.OutlineWidth != 6 || req.OutlineColor != "#112233" ||
		filepath.Base(req.OutputDir) != "output" {
		t.Fatalf("request = %+v", req)
	}
	if exists(filepath.Dir(req.OutputDir)) {
		t.Fatal("work dir survived the run")
	}
}

func clipIDOf(t *testing.T, h *harness, id string) string {
	t.Helper()
	var clipID string
	if err := h.db.QueryRow(`SELECT clip_id FROM sticker_animations WHERE id = ?`, id).Scan(&clipID); err != nil {
		t.Fatalf("read clip_id: %v", err)
	}
	return clipID
}

func TestWorkerMarksFailedWithCode(t *testing.T) {
	cases := []struct {
		name string
		proc func(h *harness) *fakeClip
		code string
	}{
		{"machine code", func(*harness) *fakeClip {
			return &fakeClip{err: &jobs.ProcessError{Code: "no_foreground", Message: "a frame has no usable foreground"}}
		}, "no_foreground"},
		{"plain failure", func(*harness) *fakeClip {
			return &fakeClip{err: &jobs.ProcessError{Message: "boom"}}
		}, jobs.CodeProcessorFailed},
		{"wrong size", func(*harness) *fakeClip {
			return &fakeClip{data: animatedWebP(256, 256), result: func(r ClipRequest) ClipResult {
				return ClipResult{Filename: fmt.Sprintf("animated/%03d.webp", r.StickerIndex), Frames: 24, FPS: 12,
					DurationMS: 1992, ByteSize: len(animatedWebP(256, 256)), Quality: 75, Width: 512, Height: 512}
			}}
		}, jobs.CodeResultInvalid},
		{"foreign filename", func(*harness) *fakeClip {
			return &fakeClip{data: animatedWebP(512, 512), result: func(r ClipRequest) ClipResult {
				return ClipResult{Filename: "../../clips/x.webp", Frames: 24, FPS: 12, DurationMS: 1992,
					ByteSize: len(animatedWebP(512, 512)), Quality: 75, Width: 512, Height: 512}
			}}
		}, jobs.CodeResultInvalid},
		{"single frame", func(*harness) *fakeClip {
			return &fakeClip{data: animatedWebP(512, 512), result: func(r ClipRequest) ClipResult {
				return ClipResult{Filename: fmt.Sprintf("animated/%03d.webp", r.StickerIndex), Frames: 1, FPS: 12,
					DurationMS: 83, ByteSize: len(animatedWebP(512, 512)), Quality: 75, Width: 512, Height: 512}
			}}
		}, jobs.CodeResultInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			packID := h.newPack(sum('a'))
			h.setDesign(packID, 4, "#FFFFFF")
			item := h.upload(packID, 0)
			stop := h.runWorker(tc.proc(h), &sync.Mutex{})
			h.waitStatus(item.ID, StatusFailed)
			stop()
			if _, code := h.status(item.ID); code != tc.code {
				t.Fatalf("error_code = %q, want %q", code, tc.code)
			}
			if exists(filepath.Join(h.dir, "animations", item.ID+".webp")) {
				t.Fatal("failed run left an animated file")
			}
		})
	}
}

func TestWorkerFailsWhenPackHasNoDesign(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'))
	item := h.upload(packID, 0)
	fake := &fakeClip{data: animatedWebP(512, 512)}
	stop := h.runWorker(fake, &sync.Mutex{})
	h.waitStatus(item.ID, StatusFailed)
	stop()
	if _, code := h.status(item.ID); code != CodeInvalidRequest || fake.callCount() != 0 {
		t.Fatalf("code = %q, calls = %d", code, fake.callCount())
	}
}

func TestWorkerNeverRunsConcurrentlyWithSheetJob(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'))
	h.setDesign(packID, 4, "#FFFFFF")
	item := h.upload(packID, 0)
	lock := &sync.Mutex{}
	lock.Lock() // a sheet job child is running
	fake := &fakeClip{data: animatedWebP(512, 512)}
	stop := h.runWorker(fake, lock)
	time.Sleep(200 * time.Millisecond)
	if fake.callCount() != 0 {
		t.Fatal("clip child started while the sheet child held the lock")
	}
	if status, _ := h.status(item.ID); status != string(StatusQueued) {
		t.Fatalf("row is %s while waiting for the lock, want queued", status)
	}
	lock.Unlock()
	h.waitStatus(item.ID, StatusReady)
	stop()
	if fake.callCount() != 1 {
		t.Fatalf("calls = %d", fake.callCount())
	}
}

func TestUploadWakesIdleWorker(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'))
	h.setDesign(packID, 4, "#FFFFFF")
	stop := h.runWorker(&fakeClip{data: animatedWebP(512, 512)}, &sync.Mutex{})
	time.Sleep(50 * time.Millisecond) // worker is idle on an empty queue
	item := h.upload(packID, 0)
	h.waitStatus(item.ID, StatusReady)
	stop()
}

func TestStartupRequeuesRunningRows(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'), sum('b'))
	running := h.upload(packID, 0)
	interrupted := h.upload(packID, 1)
	h.exec(`UPDATE sticker_animations SET status = 'running', started_at = '2026-01-01T00:00:00Z' WHERE id = ?`, running.ID)
	h.exec(`UPDATE sticker_animations SET status = 'interrupted', error_code = 'shutdown' WHERE id = ?`, interrupted.ID)

	if err := h.service.Recover(context.Background()); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	for _, id := range []string{running.ID, interrupted.ID} {
		var startedAt *string
		var code string
		var status string
		if err := h.db.QueryRow(`SELECT status, error_code, started_at FROM sticker_animations WHERE id = ?`, id).
			Scan(&status, &code, &startedAt); err != nil {
			t.Fatal(err)
		}
		if status != string(StatusQueued) || code != "" || startedAt != nil {
			t.Fatalf("row %s = %s/%q/%v, want queued and cleared", id, status, code, startedAt)
		}
	}
}

func TestRecoverPrunesOrphanDraftFiles(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'))
	item := h.upload(packID, 0)
	clip := h.clipFile(item.ID)
	animated := filepath.Join(h.dir, "animations", item.ID+".webp")
	orphanClip := filepath.Join(h.dir, "clips", uuid.NewString()+".mp4")
	orphanTmp := filepath.Join(h.dir, "clips", "."+uuid.NewString()+".tmp")
	orphanAnimated := filepath.Join(h.dir, "animations", uuid.NewString()+".webp")
	outside := filepath.Join(h.dir, "packs", "keep.webp")
	link := filepath.Join(h.dir, "animations", uuid.NewString()+".webp")
	subdir := filepath.Join(h.dir, "clips", "nested")
	for _, p := range []string{animated, orphanClip, orphanTmp, orphanAnimated, outside} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(subdir, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := h.service.Recover(context.Background()); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	for _, p := range []string{clip, animated, outside, link, subdir} {
		if !exists(p) {
			t.Fatalf("%s was removed", p)
		}
	}
	for _, p := range []string{orphanClip, orphanTmp, orphanAnimated} {
		if exists(p) {
			t.Fatalf("orphan %s survived", p)
		}
	}
}

func TestRecoverRefusesSymlinkedDraftDir(t *testing.T) {
	h := newHarness(t)
	victim := filepath.Join(h.dir, "packs", "victim.webp")
	if err := os.WriteFile(victim, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	animations := filepath.Join(h.dir, "animations")
	if err := os.Remove(animations); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(h.dir, "packs"), animations); err != nil {
		t.Fatal(err)
	}
	_ = h.service.Recover(context.Background())
	if !exists(victim) {
		t.Fatal("prune followed a symlinked animations dir")
	}
}

func TestLateFinishDoesNotTouchReplacementRow(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'))
	h.setDesign(packID, 4, "#FFFFFF")
	old := h.upload(packID, 0)
	replacement := uuid.NewString()
	fake := &fakeClip{data: animatedWebP(512, 512), hook: func(r ClipRequest) {
		// Simulate a replacement committed while the child ran (the row id changes).
		h.exec(`UPDATE sticker_animations SET id = ?, status = 'queued' WHERE id = ?`, replacement, r.AnimationID)
	}}
	// Process exactly one claim so the replacement row is never claimed.
	ctx := context.Background()
	claimed, err := h.service.claim(ctx)
	if err != nil || claimed == nil {
		t.Fatalf("claim: %v %v", claimed, err)
	}
	h.service.execute(ctx, fake, *claimed)

	if status, code := h.status(replacement); status != string(StatusQueued) || code != "" {
		t.Fatalf("replacement row = %s/%s, want untouched queued", status, code)
	}
	if exists(filepath.Join(h.dir, "animations", old.ID+".webp")) {
		t.Fatal("late finish left animations/<old id>.webp")
	}
}

// blockingClip waits for the run to end and reports why.
type blockingClip struct{ started chan struct{} }

func (b blockingClip) ProcessClip(ctx context.Context, _ ClipRequest) (ClipResult, error) {
	close(b.started)
	<-ctx.Done()
	return ClipResult{}, ctx.Err()
}

func TestWorkerTimeoutFailsAndShutdownInterrupts(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		h := newHarness(t)
		packID := h.newPack(sum('a'))
		h.setDesign(packID, 4, "#FFFFFF")
		item := h.upload(packID, 0)
		h.service.timeout = 50 * time.Millisecond
		stop := h.runWorker(blockingClip{started: make(chan struct{})}, &sync.Mutex{})
		h.waitStatus(item.ID, StatusFailed)
		stop()
		if _, code := h.status(item.ID); code != jobs.CodeTimeout {
			t.Fatalf("code = %q, want %q", code, jobs.CodeTimeout)
		}
	})
	t.Run("shutdown", func(t *testing.T) {
		h := newHarness(t)
		packID := h.newPack(sum('a'))
		h.setDesign(packID, 4, "#FFFFFF")
		item := h.upload(packID, 0)
		started := make(chan struct{})
		stop := h.runWorker(blockingClip{started: started}, &sync.Mutex{})
		<-started
		stop()
		if status, code := h.status(item.ID); status != string(StatusInterrupted) || code != jobs.CodeShutdown {
			t.Fatalf("row = %s/%s, want interrupted/%s", status, code, jobs.CodeShutdown)
		}
		// The next start (a fresh Service: Recover runs once per Service) resumes it.
		if err := NewService(h.db, h.service.files).Recover(context.Background()); err != nil {
			t.Fatal(err)
		}
		if status, code := h.status(item.ID); status != string(StatusQueued) || code != "" {
			t.Fatalf("after restart row = %s/%s, want queued", status, code)
		}
	})
}
