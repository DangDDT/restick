package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/google/uuid"
)

// Lossless animated WebP (VP8X + ANIM + 2 ANMF frames, 512x512, 100 ms each),
// the same Pillow fixture as assets/bundle_test.go.
const animated512WebP = "UklGRoQAAABXRUJQVlA4WAoAAAACAAAA/wEA/wEAQU5JTQYAAAAAAAAAAABBTk1GKAAAAAAAAAAAAP8BAP8BAGQAAAJWUDhMDwAAAC//wX8ABxD9j/4HIqL/AQBBTk1GKAAAAAAAAAAAAP8BAP8BAGQAAABWUDhMDwAAAC//wX8ABxDR//4HIqL/AQA="

// fakeAnimations stands in for animations.Service, which imports jobs.
type fakeAnimations struct {
	mu   sync.Mutex
	busy bool
	list []PublishableAnimation
}

func (f *fakeAnimations) Busy(context.Context, string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.busy, nil
}

func (f *fakeAnimations) Publishable(context.Context, string) ([]PublishableAnimation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]PublishableAnimation(nil), f.list...), nil
}

func (f *fakeAnimations) set(list ...PublishableAnimation) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.list = list
}

// publishEnv is a ready pack at v1 built by one sheet job; jobs are executed
// synchronously with claim+execute so no worker goroutine is involved.
type publishEnv struct {
	t         *testing.T
	h         *harness
	runner    *Runner
	source    *fakeAnimations
	processor *fakeProcessor
	forbid    atomic.Bool
	packID    string
	designID  string
	sourceID  string
	v1        Job
}

func newPublishEnv(t *testing.T, slug string) *publishEnv {
	t.Helper()
	e := &publishEnv{t: t, h: newHarness(t), source: &fakeAnimations{}}
	e.processor = &fakeProcessor{process: func(_ context.Context, request JobRequest) (JobResult, error) {
		if e.forbid.Load() {
			t.Errorf("processor invoked for job %s (kind %q)", request.JobID, request.Kind)
			return JobResult{}, errors.New("processor is forbidden")
		}
		return writeProcessedOutput(t, request, request.Rows*request.Columns)
	}}
	e.runner = NewRunner(e.h.db, e.h.store, e.processor, 10*time.Second, WithAnimations(e.source))
	e.packID, e.designID, e.sourceID = e.h.newPack(slug)
	job, err := e.runner.Enqueue(context.Background(), e.packID, e.sourceID, e.designID)
	if err != nil {
		t.Fatalf("enqueue sheet job: %v", err)
	}
	e.v1 = e.runOne(job.ID, StatusSucceeded)
	e.forbid.Store(true)
	return e
}

func (e *publishEnv) runOne(id string, want Status) Job {
	e.t.Helper()
	ctx := context.Background()
	claimed, err := e.runner.claim(ctx)
	if err != nil || claimed == nil || claimed.ID != id {
		e.t.Fatalf("claim = %+v, %v; want job %s", claimed, err, id)
	}
	e.runner.execute(ctx, *claimed)
	job, err := e.runner.Get(ctx, id)
	if err != nil {
		e.t.Fatalf("get job: %v", err)
	}
	if job.Status != want {
		e.t.Fatalf("job %s is %s (%s: %s), want %s", id, job.Status, job.ErrorCode, job.ErrorMessage, want)
	}
	return job
}

// animation stores data as a ready draft file, inserts the matching
// ready/enabled draft row at the current version and describes it as
// publishable. EnqueuePublish re-reads that row inside its transaction, so the
// row and the returned description must agree.
func (e *publishEnv) animation(index int, data []byte) PublishableAnimation {
	e.t.Helper()
	id := uuid.NewString()
	path, err := e.h.store.SaveAnimated(id, data)
	if err != nil {
		e.t.Fatalf("SaveAnimated: %v", err)
	}
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])
	clipID := uuid.NewString()
	if _, err := e.h.db.Exec(`INSERT INTO sticker_animations
		(id, pack_id, sticker_index, static_checksum_sha256, clip_id, clip_path, clip_byte_size,
		 clip_checksum_sha256, status, animated_path, animated_checksum_sha256, animated_byte_size,
		 frames, fps, duration_ms, quality, width, height, use_animated)
		VALUES (?, ?, ?, (SELECT checksum_sha256 FROM stickers WHERE pack_id = ? AND
			version = (SELECT current_version FROM packs WHERE id = ?) AND sort_order = ?),
			?, ?, 1, ?, 'ready', ?, ?, ?, 2, 12, 200, 75, ?, ?, 1)`,
		id, e.packID, index, e.packID, e.packID, index,
		clipID, "clips/"+clipID+".webp", strings.Repeat("c", 64), path, checksum, len(data),
		assets.StickerDimension, assets.StickerDimension); err != nil {
		e.t.Fatalf("insert draft: %v", err)
	}
	return PublishableAnimation{
		StickerIndex: index, Path: path, Checksum: checksum,
		Frames: 2, DurationMS: 200, Width: assets.StickerDimension, Height: assets.StickerDimension,
	}
}

// toggle sets use_animated on the draft row, as animations.SetUse would.
func (e *publishEnv) toggle(index int, use bool) {
	e.t.Helper()
	if _, err := e.h.db.Exec(`UPDATE sticker_animations SET use_animated = ? WHERE pack_id = ? AND sticker_index = ?`,
		use, e.packID, index); err != nil {
		e.t.Fatalf("toggle draft: %v", err)
	}
}

func (e *publishEnv) publish() Job {
	e.t.Helper()
	job, err := e.runner.EnqueuePublish(context.Background(), e.packID)
	if err != nil {
		e.t.Fatalf("EnqueuePublish: %v", err)
	}
	return e.runOne(job.ID, StatusSucceeded)
}

func (e *publishEnv) version(v int) packs.Version {
	e.t.Helper()
	version, err := e.h.catalog.GetVersion(context.Background(), e.packID, v)
	if err != nil {
		e.t.Fatalf("GetVersion %d: %v", v, err)
	}
	return version
}

type stickerRow struct {
	objectPath, thumbPath, checksum, emoji, keywords string
	animatedPath, animatedChecksum                   string
	animatedSize                                     int64
	frames, duration                                 int
}

func (e *publishEnv) stickers(v int) []stickerRow {
	e.t.Helper()
	rows, err := e.h.db.Query(`SELECT object_path, thumb_path, checksum_sha256, emoji_hint, keywords_json,
		animated_path, animated_checksum_sha256, animated_byte_size, animated_frames, animated_duration_ms
		FROM stickers WHERE pack_id = ? AND version = ? ORDER BY sort_order`, e.packID, v)
	if err != nil {
		e.t.Fatalf("read stickers: %v", err)
	}
	defer rows.Close()
	var out []stickerRow
	for rows.Next() {
		var s stickerRow
		if err := rows.Scan(&s.objectPath, &s.thumbPath, &s.checksum, &s.emoji, &s.keywords,
			&s.animatedPath, &s.animatedChecksum, &s.animatedSize, &s.frames, &s.duration); err != nil {
			e.t.Fatalf("scan sticker: %v", err)
		}
		out = append(out, s)
	}
	return out
}

func (e *publishEnv) jobCount() int {
	e.t.Helper()
	var n int
	if err := e.h.db.QueryRow(`SELECT COUNT(*) FROM processing_jobs WHERE pack_id = ?`, e.packID).Scan(&n); err != nil {
		e.t.Fatalf("count jobs: %v", err)
	}
	return n
}

func (e *publishEnv) readFile(relative string) []byte {
	e.t.Helper()
	f, err := e.h.store.OpenCommitted(relative)
	if err != nil {
		e.t.Fatalf("open %s: %v", relative, err)
	}
	defer f.Close()
	data, err := os.ReadFile(f.Name())
	if err != nil {
		e.t.Fatalf("read %s: %v", relative, err)
	}
	return data
}

func TestPublishCreatesNextVersionWithAnimatedStickers(t *testing.T) {
	e := newPublishEnv(t, "publish-next")
	if _, err := e.h.db.Exec(`UPDATE stickers SET emoji_hint = '😀', keywords_json = '["hi","wave"]'
		WHERE pack_id = ? AND version = 1 AND sort_order = 2`, e.packID); err != nil {
		t.Fatalf("set emoji: %v", err)
	}
	anim := e.animation(1, webpBytes(t, animated512WebP))
	e.source.set(anim)

	job := e.publish()
	if job.Version != e.v1.Version+1 || job.Request.Kind != KindPublish || job.Request.SchemaVersion != RequestSchemaVersion {
		t.Fatalf("publish job = v%d kind %q schema %d", job.Version, job.Request.Kind, job.Request.SchemaVersion)
	}
	pack, err := e.h.catalog.Get(context.Background(), e.packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != packs.StatusReady || pack.CurrentVersion != job.Version {
		t.Fatalf("pack is %s v%d, want ready v%d", pack.Status, pack.CurrentVersion, job.Version)
	}
	rows := e.stickers(job.Version)
	if len(rows) != 4 {
		t.Fatalf("v2 has %d stickers, want 4", len(rows))
	}
	for i, row := range rows {
		if i == 1 {
			want := "packs/" + e.packID + "/v2/animated/001.webp"
			if row.animatedPath != want || row.animatedChecksum != anim.Checksum || row.frames != 2 ||
				row.duration != 200 || row.animatedSize != int64(len(webpBytes(t, animated512WebP))) {
				t.Fatalf("animated sticker row = %+v, want %s %s", row, want, anim.Checksum)
			}
			continue
		}
		if row.animatedPath != "" || row.animatedChecksum != "" {
			t.Fatalf("sticker %d unexpectedly animated: %+v", i, row)
		}
	}
	if rows[2].emoji != "😀" || rows[2].keywords != `["hi","wave"]` {
		t.Fatalf("emoji/keywords not carried over: %+v", rows[2])
	}
	manifest := string(e.version(job.Version).ManifestJSON)
	for _, want := range []string{`"animated/001.webp"`, `"slug": "publish-next"`, `"name": "Pack publish-next"`, `"version": 2`, `"demo"`} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("pack.json lacks %s:\n%s", want, manifest)
		}
	}
	if got := e.readFile("packs/" + e.packID + "/v2/animated/001.webp"); string(got) != string(webpBytes(t, animated512WebP)) {
		t.Fatal("committed animated file differs from the draft")
	}
}

func TestPublishKeepsStaticFilesAndThumbsIdentical(t *testing.T) {
	e := newPublishEnv(t, "publish-identical")
	e.source.set(e.animation(0, webpBytes(t, animated512WebP)), e.animation(3, webpBytes(t, animated512WebP)))
	job := e.publish()

	before, after := e.stickers(e.v1.Version), e.stickers(job.Version)
	if len(before) != len(after) {
		t.Fatalf("sticker count %d -> %d", len(before), len(after))
	}
	for i := range before {
		if before[i].checksum != after[i].checksum {
			t.Fatalf("sticker %d checksum %s -> %s", i, before[i].checksum, after[i].checksum)
		}
		if string(e.readFile(before[i].objectPath)) != string(e.readFile(after[i].objectPath)) {
			t.Fatalf("sticker %d static bytes changed", i)
		}
		if string(e.readFile(before[i].thumbPath)) != string(e.readFile(after[i].thumbPath)) {
			t.Fatalf("sticker %d thumb bytes changed", i)
		}
	}
	if string(e.readFile(e.version(1).ThumbnailPath)) != string(e.readFile(e.version(job.Version).ThumbnailPath)) {
		t.Fatal("pack thumbnail changed")
	}
}

func TestPublishWithNoChangesReturnsErrNoChanges(t *testing.T) {
	e := newPublishEnv(t, "publish-nochange")
	ctx := context.Background()
	jobs := e.jobCount()
	if _, err := e.runner.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrNoChanges) {
		t.Fatalf("static pack with no drafts err = %v, want ErrNoChanges", err)
	}
	anim := e.animation(2, webpBytes(t, animated512WebP))
	e.source.set(anim)
	job := e.publish()

	// The same draft again (toggled off and back on) is no change.
	e.source.set(anim)
	jobs = e.jobCount()
	if _, err := e.runner.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrNoChanges) {
		t.Fatalf("unchanged animated set err = %v, want ErrNoChanges", err)
	}
	if got := e.jobCount(); got != jobs {
		t.Fatalf("ErrNoChanges wrote a job row: %d -> %d", jobs, got)
	}
	// No version was consumed: the next real change takes the next number.
	e.source.set(anim, e.animation(0, webpBytes(t, animated512WebP)))
	next := e.publish()
	if next.Version != job.Version+1 {
		t.Fatalf("next publish is v%d, want v%d", next.Version, job.Version+1)
	}
}

func TestPublishAllTogglesOffAfterAnimatedVersionCreatesVersionWithoutAnimated(t *testing.T) {
	e := newPublishEnv(t, "publish-off")
	e.source.set(e.animation(1, webpBytes(t, animated512WebP)))
	animated := e.publish()

	e.source.set()
	e.toggle(1, false)
	job := e.publish()
	if job.Version != animated.Version+1 {
		t.Fatalf("toggle-off publish is v%d, want v%d", job.Version, animated.Version+1)
	}
	for i, row := range e.stickers(job.Version) {
		if row.animatedPath != "" || row.animatedChecksum != "" {
			t.Fatalf("sticker %d still animated: %+v", i, row)
		}
	}
	if manifest := string(e.version(job.Version).ManifestJSON); strings.Contains(manifest, "animated") {
		t.Fatalf("pack.json still declares animated stickers:\n%s", manifest)
	}
	if _, err := e.runner.EnqueuePublish(context.Background(), e.packID); !errors.Is(err, ErrNoChanges) {
		t.Fatalf("second toggle-off err = %v, want ErrNoChanges", err)
	}
}

func TestPublishBlockedWhileClipRunning(t *testing.T) {
	e := newPublishEnv(t, "publish-busy")
	e.source.set(e.animation(1, webpBytes(t, animated512WebP)))
	e.source.mu.Lock()
	e.source.busy = true
	e.source.mu.Unlock()
	jobs := e.jobCount()
	if _, err := e.runner.EnqueuePublish(context.Background(), e.packID); !errors.Is(err, ErrAnimationBusy) {
		t.Fatalf("err = %v, want ErrAnimationBusy", err)
	}
	if got := e.jobCount(); got != jobs {
		t.Fatalf("busy publish wrote a job row: %d -> %d", jobs, got)
	}
}

func TestPublishBlockedWhileSheetJobActive(t *testing.T) {
	e := newPublishEnv(t, "publish-active")
	ctx := context.Background()
	e.source.set(e.animation(1, webpBytes(t, animated512WebP)))
	if _, err := e.runner.Enqueue(ctx, e.packID, e.sourceID, e.designID); err != nil {
		t.Fatalf("enqueue sheet job: %v", err)
	}
	if _, err := e.runner.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrJobActive) {
		t.Fatalf("queued sheet job err = %v, want ErrJobActive", err)
	}
	// Claimed but not executed: the sheet job is running.
	if claimed, err := e.runner.claim(ctx); err != nil || claimed == nil {
		t.Fatalf("claim: %+v %v", claimed, err)
	}
	if _, err := e.runner.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrJobActive) {
		t.Fatalf("running sheet job err = %v, want ErrJobActive", err)
	}
}

// racingAnimations is the deterministic race seam: it returns the drafts read
// before EnqueuePublish opens its transaction and then mutates the database, as
// a concurrent upload or toggle would.
type racingAnimations struct {
	source *fakeAnimations
	after  func()
}

func (r racingAnimations) Busy(ctx context.Context, packID string) (bool, error) {
	return r.source.Busy(ctx, packID)
}

func (r racingAnimations) Publishable(ctx context.Context, packID string) ([]PublishableAnimation, error) {
	list, err := r.source.Publishable(ctx, packID)
	r.after()
	return list, err
}

func TestPublishConflictsWhenDraftsChangeAfterTheRead(t *testing.T) {
	// Either a list that gained a draft or one that lost it would queue a
	// publication the request never approved (a silently dropped or added
	// animation), so both are a conflict and no job row is written.
	cases := map[string]struct {
		prepare func(e *publishEnv)
		after   func(e *publishEnv)
	}{
		"draft appears after the read": {
			prepare: func(e *publishEnv) {},
			after: func(e *publishEnv) {
				e.animation(2, webpBytes(t, animated512WebP))
			},
		},
		"draft toggled off after the read": {
			prepare: func(e *publishEnv) {
				e.source.set(e.animation(1, webpBytes(t, animated512WebP)))
			},
			after: func(e *publishEnv) { e.toggle(1, false) },
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := newPublishEnv(t, "publish-race")
			ctx := context.Background()
			tc.prepare(e)
			jobs := e.jobCount()
			e.runner = NewRunner(e.h.db, e.h.store, e.processor, 10*time.Second,
				WithAnimations(racingAnimations{source: e.source, after: func() { tc.after(e) }}))

			if _, err := e.runner.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrAnimationBusy) {
				t.Fatalf("err = %v, want ErrAnimationBusy", err)
			}
			if got := e.jobCount(); got != jobs {
				t.Fatalf("conflicting publish wrote a job row: %d -> %d", jobs, got)
			}
			pack, err := e.h.catalog.Get(ctx, e.packID)
			if err != nil {
				t.Fatalf("get pack: %v", err)
			}
			if pack.CurrentVersion != e.v1.Version {
				t.Fatalf("pack moved to v%d after a conflicting publish", pack.CurrentVersion)
			}
		})
	}
}

func TestPublishRequiresLivePackWithVersion(t *testing.T) {
	e := newPublishEnv(t, "publish-guards")
	ctx := context.Background()
	if _, err := e.runner.EnqueuePublish(ctx, uuid.NewString()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing pack err = %v, want ErrNotFound", err)
	}
	draftPack, _, _ := e.h.newPack("publish-draft")
	if _, err := e.runner.EnqueuePublish(ctx, draftPack); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pack without version err = %v, want ErrNotFound", err)
	}
	e.source.set(e.animation(1, webpBytes(t, animated512WebP)))
	if err := e.runner.Revoke(ctx, e.packID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := e.runner.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrPackRevoked) {
		t.Fatalf("revoked pack err = %v, want ErrPackRevoked", err)
	}
	unwired := NewRunner(e.h.db, e.h.store, nil, time.Second)
	if _, err := unwired.EnqueuePublish(ctx, e.packID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("runner without animations err = %v, want ErrInvalid", err)
	}
}

func TestPublishDoesNotInvokeProcessor(t *testing.T) {
	e := newPublishEnv(t, "publish-noproc")
	// A deleted source sheet does not matter: publish never reads it.
	if _, err := e.h.db.Exec(`UPDATE packs SET sheet_id = NULL WHERE id = ?`, e.packID); err != nil {
		t.Fatalf("detach sheet: %v", err)
	}
	e.source.set(e.animation(1, webpBytes(t, animated512WebP)))
	e.publish()
	e.processor.mu.Lock()
	calls := len(e.processor.calls)
	e.processor.mu.Unlock()
	if calls != 1 {
		t.Fatalf("processor ran %d times, want only the v1 sheet job", calls)
	}
}

func TestPublishFailureKeepsOldVersionReady(t *testing.T) {
	cases := map[string]func(e *publishEnv) PublishableAnimation{
		// A single-frame file is rejected by StageBundle's animated check.
		"not animated": func(e *publishEnv) PublishableAnimation {
			return e.animation(1, webpBytes(t, stickerWebP))
		},
		// The draft row records a digest the draft file does not have.
		"checksum mismatch": func(e *publishEnv) PublishableAnimation {
			anim := e.animation(1, webpBytes(t, animated512WebP))
			anim.Checksum = strings.Repeat("0", 64)
			if _, err := e.h.db.Exec(`UPDATE sticker_animations SET animated_checksum_sha256 = ?
				WHERE pack_id = ? AND sticker_index = 1`, anim.Checksum, e.packID); err != nil {
				e.t.Fatalf("corrupt draft checksum: %v", err)
			}
			return anim
		},
	}
	for name, makeAnim := range cases {
		t.Run(name, func(t *testing.T) {
			e := newPublishEnv(t, "publish-fail")
			ctx := context.Background()
			e.source.set(makeAnim(e))
			job, err := e.runner.EnqueuePublish(ctx, e.packID)
			if err != nil {
				t.Fatalf("EnqueuePublish: %v", err)
			}
			failed := e.runOne(job.ID, StatusFailed)
			if failed.ErrorCode != CodeStageFailed {
				t.Fatalf("error code = %q (%s), want %q", failed.ErrorCode, failed.ErrorMessage, CodeStageFailed)
			}
			pack, err := e.h.catalog.Get(ctx, e.packID)
			if err != nil {
				t.Fatalf("get pack: %v", err)
			}
			if pack.Status != packs.StatusReady || pack.CurrentVersion != e.v1.Version {
				t.Fatalf("pack is %s v%d, want ready v%d", pack.Status, pack.CurrentVersion, e.v1.Version)
			}
			if _, err := e.h.catalog.GetVersion(ctx, e.packID, job.Version); !errors.Is(err, packs.ErrVersionNotFound) {
				t.Fatalf("failed version err = %v, want ErrVersionNotFound", err)
			}
			if _, err := os.Lstat(filepath.Join(filepath.Dir(e.readFileDir()), "v2")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("half version directory exists: %v", err)
			}
		})
	}
}

// readFileDir returns the absolute packs/<id>/v1 directory.
func (e *publishEnv) readFileDir() string {
	e.t.Helper()
	f, err := e.h.store.OpenCommitted("packs/" + e.packID + "/v1/pack.json")
	if err != nil {
		e.t.Fatalf("open v1 pack.json: %v", err)
	}
	defer f.Close()
	return filepath.Dir(f.Name())
}

func TestPublishRepublishKeepsSuggestedThemeColor(t *testing.T) {
	e := newPublishEnv(t, "publish-keep-color")
	if _, err := e.h.db.Exec(`UPDATE packs SET theme_color_suggested = '#A98BF0' WHERE id = ?`, e.packID); err != nil {
		t.Fatal(err)
	}
	e.source.set(e.animation(0, webpBytes(t, animated512WebP)))
	job := e.publish()
	pack, err := e.h.catalog.Get(context.Background(), e.packID)
	if err != nil {
		t.Fatal(err)
	}
	if pack.CurrentVersion != job.Version || pack.ThemeColorSuggested != "#A98BF0" {
		t.Fatalf("v%d suggested=%q, want republish v%d keeping #A98BF0", pack.CurrentVersion, pack.ThemeColorSuggested, job.Version)
	}
}
