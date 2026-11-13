package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/animations"
	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/google/uuid"
	"golang.org/x/image/webp"
)

// clipScript writes a one-second 24 fps mp4: a red disc drifting over white,
// the same synthetic clip media_factory's own clip_job tests use.
const clipScript = `
import sys, cv2, numpy as np
w = cv2.VideoWriter(sys.argv[1], cv2.VideoWriter_fourcc(*"mp4v"), 24, (512, 512))
if not w.isOpened():
    sys.exit(3)
for i in range(24):
    f = np.full((512, 512, 3), 255, np.uint8)
    cv2.circle(f, (256 + i, 256), 128, (40, 40, 220), -1)
    w.write(f)
w.release()
`

// TestClipRoundTrip runs the real PythonClipProcessor through the clip queue:
// upload, worker, verified animated WebP at animations/<id>.webp.
func TestClipRoundTrip(t *testing.T) {
	if err := exec.Command("python", "-c", "import media_factory, cv2").Run(); err != nil {
		t.Skipf("python with media_factory and cv2 unavailable: %v", err)
	}
	dir := t.TempDir()
	clip := filepath.Join(dir, "clip.mp4")
	if out, err := exec.Command("python", "-c", clipScript, clip).CombinedOutput(); err != nil {
		t.Skipf("cannot write a synthetic mp4 clip: %v %s", err, out)
	}
	clipBytes, err := os.ReadFile(clip)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	dataDir := filepath.Join(dir, "data")
	db, err := storage.Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := assets.NewStore(dataDir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	packID := seedClipPack(t, db)
	service := animations.NewService(db, files)
	item, err := service.Upload(ctx, packID, 0, bytes.NewReader(clipBytes), "clip.mp4", "drift right")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- service.RunWorker(runCtx, animations.PythonClipProcessor{}, &sync.Mutex{}) }()
	var got animations.Animation
	deadline := time.Now().Add(processorTimeout)
	for time.Now().Before(deadline) {
		list, err := service.List(ctx, packID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got = list[0]
		if got.Status != animations.StatusQueued && got.Status != animations.StatusRunning {
			break
		}
		time.Sleep(pollInterval)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("RunWorker: %v", err)
	}
	if got.Status != animations.StatusReady {
		t.Fatalf("clip ended %s: %s %s", got.Status, got.ErrorCode, got.ErrorMessage)
	}
	if got.AnimatedPath != "animations/"+item.ID+".webp" || got.Frames < 2 || got.FPS != 12 ||
		got.Width != 512 || got.Height != 512 || got.AnimatedByteSize > 512_000 {
		t.Fatalf("ready animation = %+v", got)
	}
	data, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(got.AnimatedPath)))
	if err != nil {
		t.Fatalf("read animated file: %v", err)
	}
	if int64(len(data)) != got.AnimatedByteSize || !strings.HasPrefix(string(data[8:16]), "WEBPVP8X") {
		t.Fatalf("animated file is %d bytes, header %q", len(data), data[:16])
	}
	cfg, err := webp.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width != 512 || cfg.Height != 512 {
		t.Fatalf("DecodeConfig = %+v, %v", cfg, err)
	}
	if entries, _ := os.ReadDir(filepath.Join(dataDir, "work")); len(entries) != 0 {
		t.Fatalf("clip work dir survived: %v", entries)
	}
}

// seedClipPack inserts a pack with a committed one-sticker version and a
// design, the minimum a clip upload and its outline lookup need.
func seedClipPack(t *testing.T, db *sql.DB) string {
	t.Helper()
	packID, jobID, designID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	sum := strings.Repeat("a", 64)
	for _, stmt := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO designs (id, name, rows, columns, outline_width, outline_color, draft_json, snapshot_json)
			VALUES (?, 'D', 1, 1, 4, '#FFFFFF', '{}', '{"schema_version":1}')`, []any{designID}},
		{`INSERT INTO packs (id, slug, name, author, design_id) VALUES (?, ?, 'Pack', 'Restick', ?)`, []any{packID, packID, designID}},
		{`INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES (?, ?, 1, 'succeeded', '{"schema_version":1}')`, []any{jobID, packID}},
		{`INSERT INTO pack_versions (pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
			VALUES (?, 1, ?, ?, 1, '{}', 'b', 1, 't', 'o')`, []any{packID, jobID, sum}},
		{`INSERT INTO stickers (pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height)
			VALUES (?, 1, 0, '000.webp', 'o', 't', ?, 512, 512)`, []any{packID, sum}},
		{`UPDATE packs SET current_version = 1, status = 'ready' WHERE id = ?`, []any{packID}},
	} {
		if _, err := db.Exec(stmt.query, stmt.args...); err != nil {
			t.Fatalf("seed %s: %v", fmt.Sprint(stmt.query[:30]), err)
		}
	}
	return packID
}
