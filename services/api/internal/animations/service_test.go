package animations

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/google/uuid"
)

var mp4Clip = append([]byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, bytes.Repeat([]byte{7}, 64)...)

type harness struct {
	t       *testing.T
	dir     string
	db      *sql.DB
	service *Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := assets.NewStore(dir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	return &harness{t: t, dir: dir, db: db, service: NewService(db, files)}
}

func (h *harness) exec(query string, args ...any) {
	h.t.Helper()
	if _, err := h.db.Exec(query, args...); err != nil {
		h.t.Fatalf("exec %q: %v", query, err)
	}
}

func sum(c byte) string { return strings.Repeat(string(c), 64) }

// addVersion commits version v of the pack with one sticker per checksum and
// makes it current.
func (h *harness) addVersion(packID string, v int, checksums ...string) {
	h.t.Helper()
	jobID := uuid.NewString()
	h.exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES (?, ?, ?, 'succeeded', '{"schema_version":1}')`, jobID, packID, v)
	h.exec(`INSERT INTO pack_versions (pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, ?, ?, ?, ?, '{}', 'b', 1, 't', 'o')`, packID, v, jobID, sum('f'), len(checksums))
	for i, c := range checksums {
		h.exec(`INSERT INTO stickers (pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height)
			VALUES (?, ?, ?, ?, 'o', 't', ?, 512, 512)`, packID, v, i, fmt.Sprintf("%03d.webp", i), c)
	}
	h.exec(`UPDATE packs SET current_version = ?, status = 'ready' WHERE id = ?`, v, packID)
}

func (h *harness) newPack(checksums ...string) string {
	h.t.Helper()
	id := uuid.NewString()
	h.exec(`INSERT INTO packs (id, slug, name, author) VALUES (?, ?, 'Pack', 'Restick')`, id, id)
	h.addVersion(id, 1, checksums...)
	return id
}

func (h *harness) upload(packID string, index int) Animation {
	h.t.Helper()
	item, err := h.service.Upload(context.Background(), packID, index, bytes.NewReader(mp4Clip), "clip.mp4", "vẫy tay")
	if err != nil {
		h.t.Fatalf("Upload: %v", err)
	}
	return item
}

func (h *harness) clipFile(animationID string) string {
	h.t.Helper()
	var clipPath string
	if err := h.db.QueryRow(`SELECT clip_path FROM sticker_animations WHERE id = ?`, animationID).Scan(&clipPath); err != nil {
		h.t.Fatalf("read clip_path: %v", err)
	}
	return filepath.Join(h.dir, filepath.FromSlash(clipPath))
}

func (h *harness) clipCount() int {
	h.t.Helper()
	entries, err := os.ReadDir(filepath.Join(h.dir, "clips"))
	if err != nil {
		h.t.Fatalf("read clips: %v", err)
	}
	return len(entries)
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func TestUploadCreatesQueuedRow(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'), sum('b'))

	item := h.upload(packID, 1)
	if item.Status != StatusQueued || item.PackID != packID || item.StickerIndex != 1 ||
		item.StaticChecksum != sum('b') || item.MotionText != "vẫy tay" || item.ClipByteSize != int64(len(mp4Clip)) || item.Stale {
		t.Fatalf("animation = %+v", item)
	}
	if !exists(h.clipFile(item.ID)) {
		t.Fatal("clip file missing")
	}
	items, err := h.service.List(ctx, packID)
	if err != nil || len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("List = %+v (err %v)", items, err)
	}
	if busy, err := h.service.Busy(ctx, packID); err != nil || !busy {
		t.Fatalf("Busy = %v (err %v), want true", busy, err)
	}
}

func TestUploadValidatesPackAndClip(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'))

	if _, err := h.service.Upload(ctx, uuid.NewString(), 0, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown pack = %v, want ErrNotFound", err)
	}
	if _, err := h.service.Upload(ctx, "not-a-uuid", 0, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bad pack id = %v, want ErrNotFound", err)
	}
	if _, err := h.service.Upload(ctx, packID, 0, strings.NewReader("not a video at all"), "c.mp4", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad clip = %v, want ErrInvalid", err)
	}
	if _, err := h.service.Upload(ctx, packID, 0, bytes.NewReader(mp4Clip), "c.mp4", strings.Repeat("x", 501)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("long motion text = %v, want ErrInvalid", err)
	}
	h.exec(`UPDATE packs SET status = 'revoked' WHERE id = ?`, packID)
	if _, err := h.service.Upload(ctx, packID, 0, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("revoked pack = %v, want ErrInvalid", err)
	}
	if n := h.clipCount(); n != 0 {
		t.Fatalf("rejected uploads left %d clip files", n)
	}
}

func TestReuploadReplacesRowAndRemovesOldClip(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'))

	first := h.upload(packID, 0)
	oldClip := h.clipFile(first.ID)
	oldAnimated := filepath.Join(h.dir, "animations", first.ID+".webp")
	if err := os.WriteFile(oldAnimated, []byte("webp"), 0o644); err != nil {
		t.Fatal(err)
	}

	second := h.upload(packID, 0)
	if second.ID == first.ID {
		t.Fatal("re-upload kept the old id")
	}
	if exists(oldClip) || exists(oldAnimated) {
		t.Fatal("old files were not removed")
	}
	if !exists(h.clipFile(second.ID)) {
		t.Fatal("new clip missing")
	}
	items, err := h.service.List(ctx, packID)
	if err != nil || len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("List = %+v (err %v)", items, err)
	}
}

func TestUploadRejectsIndexOutOfRange(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'), sum('b'))

	for _, index := range []int{2, -1, 99} {
		if _, err := h.service.Upload(ctx, packID, index, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrInvalid) {
			t.Errorf("index %d = %v, want ErrInvalid", index, err)
		}
	}
	draft := uuid.NewString()
	h.exec(`INSERT INTO packs (id, slug, name, author) VALUES (?, ?, 'Draft', 'Restick')`, draft, draft)
	if _, err := h.service.Upload(ctx, draft, 0, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("pack without a version = %v, want ErrInvalid", err)
	}
	if n := h.clipCount(); n != 0 {
		t.Fatalf("rejected uploads left %d clip files", n)
	}
}

// Review Focus 5: upload, delete and toggle are refused while a sheet or
// publish job is queued or running for the pack.
func TestUploadRejectedWhileSheetJobActive(t *testing.T) {
	for _, status := range []string{"queued", "running"} {
		t.Run(status, func(t *testing.T) {
			h := newHarness(t)
			ctx := context.Background()
			packID := h.newPack(sum('a'), sum('b'))
			ready := h.upload(packID, 0)
			h.exec(`UPDATE sticker_animations SET status = 'ready' WHERE id = ?`, ready.ID)
			h.exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES (?, ?, 2, ?, '{"schema_version":1}')`, uuid.NewString(), packID, status)

			if _, err := h.service.Upload(ctx, packID, 1, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrJobActive) {
				t.Fatalf("Upload = %v, want ErrJobActive", err)
			}
			if h.clipCount() != 1 {
				t.Fatal("refused upload left its clip behind")
			}
			if _, err := h.service.SetUse(ctx, packID, 0, false); !errors.Is(err, ErrJobActive) {
				t.Fatalf("SetUse = %v, want ErrJobActive", err)
			}
			if err := h.service.Delete(ctx, packID, 0); !errors.Is(err, ErrJobActive) {
				t.Fatalf("Delete = %v, want ErrJobActive", err)
			}
			items, err := h.service.List(ctx, packID)
			if err != nil || len(items) != 1 || !items[0].UseAnimated || !exists(h.clipFile(ready.ID)) {
				t.Fatalf("refused calls changed state: %+v (err %v)", items, err)
			}
		})
	}
}

func TestUploadAndDeleteRejectedWhileClipRunning(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'))
	item := h.upload(packID, 0)
	h.exec(`UPDATE sticker_animations SET status = 'running' WHERE id = ?`, item.ID)

	if _, err := h.service.Upload(ctx, packID, 0, bytes.NewReader(mp4Clip), "c.mp4", ""); !errors.Is(err, ErrJobActive) {
		t.Fatalf("Upload over running clip = %v, want ErrJobActive", err)
	}
	if err := h.service.Delete(ctx, packID, 0); !errors.Is(err, ErrJobActive) {
		t.Fatalf("Delete of running clip = %v, want ErrJobActive", err)
	}
	if h.clipCount() != 1 || !exists(h.clipFile(item.ID)) {
		t.Fatal("running clip files changed")
	}
}

func TestSetUseRequiresReady(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'))
	item := h.upload(packID, 0)

	if _, err := h.service.SetUse(ctx, packID, 0, true); !errors.Is(err, ErrNotReady) {
		t.Fatalf("SetUse on queued = %v, want ErrNotReady", err)
	}
	if _, err := h.service.SetUse(ctx, packID, 1, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetUse on missing = %v, want ErrNotFound", err)
	}
	h.exec(`UPDATE sticker_animations SET status = 'ready' WHERE id = ?`, item.ID)
	off, err := h.service.SetUse(ctx, packID, 0, false)
	if err != nil || off.UseAnimated {
		t.Fatalf("SetUse(false) = %+v (err %v)", off, err)
	}
	on, err := h.service.SetUse(ctx, packID, 0, true)
	if err != nil || !on.UseAnimated {
		t.Fatalf("SetUse(true) = %+v (err %v)", on, err)
	}
}

// Review Focus 3: a draft made against v1 is stale once the current version
// carries a different static sticker at its index.
func TestListMarksStaleWhenStaticChecksumChanged(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'), sum('b'), sum('c'))
	changed := h.upload(packID, 0)
	same := h.upload(packID, 1)
	dropped := h.upload(packID, 2)

	stale := func() map[string]bool {
		items, err := h.service.List(ctx, packID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		out := map[string]bool{}
		for _, item := range items {
			out[item.ID] = item.Stale
		}
		return out
	}
	if got := stale(); got[changed.ID] || got[same.ID] || got[dropped.ID] {
		t.Fatalf("fresh drafts flagged stale: %v", got)
	}

	// v2: index 0 redrawn, index 1 unchanged, index 2 no longer exists.
	h.addVersion(packID, 2, sum('d'), sum('b'))
	got := stale()
	if !got[changed.ID] {
		t.Error("draft over a changed sticker is not stale")
	}
	if got[same.ID] {
		t.Error("draft over an unchanged sticker is stale")
	}
	if !got[dropped.ID] {
		t.Error("draft over a removed sticker is not stale")
	}
}

func TestDeleteRemovesFiles(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	packID := h.newPack(sum('a'))
	item := h.upload(packID, 0)
	clip := h.clipFile(item.ID)
	animated := filepath.Join(h.dir, "animations", item.ID+".webp")
	if err := os.WriteFile(animated, []byte("webp"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := h.service.Delete(ctx, packID, 0); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if exists(clip) || exists(animated) {
		t.Fatal("files remain after delete")
	}
	if items, err := h.service.List(ctx, packID); err != nil || len(items) != 0 {
		t.Fatalf("List after delete = %+v (err %v)", items, err)
	}
	if err := h.service.Delete(ctx, packID, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
	if busy, err := h.service.Busy(ctx, packID); err != nil || busy {
		t.Fatalf("Busy = %v (err %v), want false", busy, err)
	}
}
