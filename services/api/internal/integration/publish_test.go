package integration

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/animations"
	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/google/uuid"
)

// The dependency is one-way (animations imports jobs), so the interface is
// asserted here rather than in either package.
var _ jobs.AnimationSource = (*animations.Service)(nil)

// Static 512/128 lossless WebP and a 2-frame 512x512 animated WebP, the same
// fixtures as the jobs and assets tests.
const (
	publishStickerWebP  = "UklGRi4AAABXRUJQVlA4TCIAAAAv/8F/AAdQlHIXpf8BgUCyv/cMRfQ/4z//+c9//vOf//wf"
	publishThumbWebP    = "UklGRhgAAABXRUJQVlA4TAsAAAAvf8AfAEVRShv9DwA="
	publishAnimatedWebP = "UklGRoQAAABXRUJQVlA4WAoAAAACAAAA/wEA/wEAQU5JTQYAAAAAAAAAAABBTk1GKAAAAAAAAAAAAP8BAP8BAGQAAAJWUDhMDwAAAC//wX8ABxD9j/4HIqL/AQBBTk1GKAAAAAAAAAAAAP8BAP8BAGQAAABWUDhMDwAAAC//wX8ABxDR//4HIqL/AQA="
)

// TestPublishExcludesStaleAnimation runs the real animations.Service as the
// runner's AnimationSource on the single-connection database: the publish job
// takes the fresh draft and leaves out the one drafted against an old static.
func TestPublishExcludesStaleAnimation(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	db, err := storage.Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := assets.NewStore(dataDir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	decode := func(encoded string) []byte {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("decode fixture: %v", err)
		}
		return data
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("exec %q: %v", query[:30], err)
		}
	}

	// v1: two static stickers committed through the normal publication path.
	packID, v1Job := uuid.NewString(), uuid.NewString()
	exec(`INSERT INTO packs (id, slug, name, author) VALUES (?, 'stale-pack', 'Stale', 'Restick')`, packID)
	exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json) VALUES (?, ?, 1, 'running', '{"schema_version":1}')`, v1Job, packID)
	workDir, err := files.WorkDir(v1Job)
	if err != nil {
		t.Fatalf("WorkDir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "output"), 0o755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	manifest := assets.PackManifest{PackID: packID, Slug: "stale-pack", Name: "Stale", Author: "Restick", Version: 1}
	for i := range 2 {
		file := fmt.Sprintf("%03d.webp", i)
		for name, encoded := range map[string]string{file: publishStickerWebP, "thumb_" + file: publishThumbWebP} {
			if err := os.WriteFile(filepath.Join(workDir, "output", name), decode(encoded), 0o644); err != nil {
				t.Fatalf("write output: %v", err)
			}
		}
		manifest.Stickers = append(manifest.Stickers, assets.ManifestSticker{
			File: file, ThumbFile: "thumb_" + file, Width: 512, Height: 512,
		})
	}
	staged, err := files.StageBundle(ctx, v1Job, manifest)
	if err != nil {
		t.Fatalf("StageBundle: %v", err)
	}
	committed, err := files.CommitFiles(ctx, staged)
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}
	if err := packs.CommitProcessed(ctx, db, packs.ProcessedVersion{
		JobID: v1Job, PackID: packID, Version: 1, Manifest: manifest, Files: committed,
	}); err != nil {
		t.Fatalf("CommitProcessed: %v", err)
	}

	// Ready drafts: sticker 0 against the current static, sticker 1 stale.
	animated := decode(publishAnimatedWebP)
	animatedSum := sha256.Sum256(animated)
	for i, static := range []string{committed.Stickers[0].ChecksumSHA256, strings.Repeat("e", 64)} {
		id := uuid.NewString()
		animatedPath, err := files.SaveAnimated(id, animated)
		if err != nil {
			t.Fatalf("SaveAnimated: %v", err)
		}
		exec(`INSERT INTO sticker_animations (id, pack_id, sticker_index, static_checksum_sha256, clip_id, clip_path,
			clip_byte_size, clip_checksum_sha256, status, animated_path, animated_checksum_sha256, animated_byte_size,
			frames, fps, duration_ms, width, height)
			VALUES (?, ?, ?, ?, ?, 'clips/x.mp4', 1, ?, 'ready', ?, ?, ?, 2, 10, 200, 512, 512)`,
			id, packID, i, static, uuid.NewString(), strings.Repeat("c", 64), animatedPath,
			hex.EncodeToString(animatedSum[:]), len(animated))
	}

	service := animations.NewService(db, files)
	// A nil processor would panic if the publish job ever started a child.
	runner := jobs.NewRunner(db, files, nil, time.Minute, jobs.WithAnimations(service))
	job, err := runner.EnqueuePublish(ctx, packID)
	if err != nil {
		t.Fatalf("EnqueuePublish: %v", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- runner.Run(runCtx) }()
	deadline := time.Now().Add(15 * time.Second)
	for {
		job, err = runner.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if job.Status != jobs.StatusQueued && job.Status != jobs.StatusRunning {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("publish job still %s", job.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if job.Status != jobs.StatusSucceeded {
		t.Fatalf("publish job %s: %s %s", job.Status, job.ErrorCode, job.ErrorMessage)
	}

	version, err := packs.NewService(db).GetVersion(ctx, packID, 2)
	if err != nil {
		t.Fatalf("GetVersion 2: %v", err)
	}
	var animatedPaths []string
	rows, err := db.Query(`SELECT animated_path FROM stickers WHERE pack_id = ? AND version = 2 ORDER BY sort_order`, packID)
	if err != nil {
		t.Fatalf("read stickers: %v", err)
	}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatalf("scan: %v", err)
		}
		animatedPaths = append(animatedPaths, p)
	}
	rows.Close()
	want := []string{"packs/" + packID + "/v2/animated/000.webp", ""}
	if fmt.Sprint(animatedPaths) != fmt.Sprint(want) {
		t.Fatalf("v2 animated paths = %q, want %q", animatedPaths, want)
	}
	if strings.Contains(string(version.ManifestJSON), "animated/001.webp") {
		t.Fatalf("stale draft published:\n%s", version.ManifestJSON)
	}
	// The stale draft alone is no change against v2.
	if _, err := runner.EnqueuePublish(ctx, packID); !errors.Is(err, jobs.ErrNoChanges) {
		t.Fatalf("republish err = %v, want ErrNoChanges", err)
	}
}
