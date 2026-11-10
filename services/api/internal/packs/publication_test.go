package packs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/google/uuid"
)

// processedVersion builds a fully written version, the way jobs prepares it
// after staging and renaming its files.
func processedVersion(packID, jobID string, version, count int) ProcessedVersion {
	outputDir := fmt.Sprintf("packs/%s/v%d", packID, version)
	manifest := assets.PackManifest{
		PackID: packID, Slug: "demo-pack", Name: "Demo Pack", Author: "Tester",
		Tags: []string{"demo"}, Version: version,
	}
	files := assets.CommittedFiles{
		JobID:          jobID,
		PackID:         packID,
		Version:        version,
		ChecksumSHA256: testChecksum("c"),
		StickerCount:   count,
		BundlePath:     fmt.Sprintf("bundles/%s/v%d.zip", packID, version),
		BundleByteSize: 4096,
		ThumbnailPath:  outputDir + "/thumb.webp",
		OutputDir:      outputDir,
	}
	for i := range count {
		file := fmt.Sprintf("%03d.webp", i)
		manifest.Stickers = append(manifest.Stickers, assets.ManifestSticker{
			File: file, ThumbFile: "thumb_" + file, Width: 512, Height: 512,
		})
		files.Stickers = append(files.Stickers, assets.CommittedSticker{
			SortOrder:      i,
			Filename:       file,
			ObjectPath:     outputDir + "/" + file,
			ThumbPath:      outputDir + "/thumbs/thumb_" + file,
			ChecksumSHA256: testChecksum("d"),
			Width:          512,
			Height:         512,
		})
	}
	manifestJSON, err := manifest.PackJSON(files.ChecksumSHA256)
	if err != nil {
		panic(err)
	}
	files.ManifestJSON = manifestJSON
	return ProcessedVersion{
		JobID:       jobID,
		PackID:      packID,
		Version:     version,
		Manifest:    manifest,
		Files:       files,
		Diagnostics: `{"engine":"fixture"}`,
	}
}

// runningJob inserts a job row the publication transaction can commit.
func runningJob(t *testing.T, db *sql.DB, service *Service, slug string, version int) (string, string) {
	t.Helper()
	pack, err := service.Create(context.Background(), PackDraft{
		Slug: slug, Name: "Pack " + slug, Author: "Tester",
	})
	if err != nil {
		t.Fatalf("create pack: %v", err)
	}
	jobID := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json, started_at)
		VALUES (?, ?, ?, 'running', '{"schema_version":1}',
			strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))`, jobID, pack.ID, version); err != nil {
		t.Fatalf("insert running job: %v", err)
	}
	return pack.ID, jobID
}

func TestCommitProcessedPublishesVersion(t *testing.T) {
	service, db := openTestService(t)
	ctx := context.Background()
	packID, jobID := runningJob(t, db, service, "publish-pack", 1)
	result := processedVersion(packID, jobID, 1, 2)

	if err := CommitProcessed(ctx, db, result); err != nil {
		t.Fatalf("CommitProcessed: %v", err)
	}
	pack, err := service.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != StatusReady || pack.CurrentVersion != 1 || pack.StickerCount != 2 {
		t.Fatalf("pack is %s v%d with %d stickers", pack.Status, pack.CurrentVersion, pack.StickerCount)
	}
	version, err := service.GetVersion(ctx, packID, 1)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if version.JobID != jobID || version.ChecksumSHA256 != result.Files.ChecksumSHA256 ||
		version.BundlePath != result.Files.BundlePath || version.ThumbnailPath != result.Files.ThumbnailPath {
		t.Fatalf("committed version rows do not match the publication: %+v", version)
	}
	if string(version.ManifestJSON) != string(result.Files.ManifestJSON) {
		t.Error("stored manifest is not the exact pack.json snapshot")
	}
	if len(version.Stickers) != 2 || version.Stickers[0].SortOrder != 0 ||
		version.Stickers[1].Filename != "001.webp" ||
		version.Stickers[1].ThumbPath != result.Files.Stickers[1].ThumbPath {
		t.Fatalf("committed stickers are not in row-major order: %+v", version.Stickers)
	}

	var status, resultJSON, diagnostics, finishedAt string
	if err := db.QueryRow(`SELECT status, result_json, diagnostics, finished_at FROM processing_jobs WHERE id = ?`,
		jobID).Scan(&status, &resultJSON, &diagnostics, &finishedAt); err != nil {
		t.Fatalf("read job: %v", err)
	}
	if status != "succeeded" || finishedAt == "" {
		t.Fatalf("job is %s finished %q", status, finishedAt)
	}
	if diagnostics != result.Diagnostics {
		t.Errorf("job diagnostics = %q", diagnostics)
	}
	var summary struct {
		Schema         string `json:"schema"`
		Version        int    `json:"version"`
		ChecksumSHA256 string `json:"checksum_sha256"`
	}
	if err := json.Unmarshal([]byte(resultJSON), &summary); err != nil {
		t.Fatalf("decode result json: %v", err)
	}
	if summary.Schema != publicationSchema || summary.Version != 1 || summary.ChecksumSHA256 != result.Files.ChecksumSHA256 {
		t.Fatalf("result json = %s", resultJSON)
	}

	if err := CommitProcessed(ctx, db, result); !errors.Is(err, ErrJobNotRunning) {
		t.Fatalf("second publication = %v, want ErrJobNotRunning", err)
	}
}

func TestCommitProcessedRequiresRunningJob(t *testing.T) {
	service, db := openTestService(t)
	ctx := context.Background()
	packID, jobID := runningJob(t, db, service, "queued-pack", 2)
	if _, err := db.Exec(`UPDATE processing_jobs SET status = 'interrupted' WHERE id = ?`, jobID); err != nil {
		t.Fatalf("interrupt job: %v", err)
	}
	result := processedVersion(packID, jobID, 2, 2)
	if err := CommitProcessed(ctx, db, result); !errors.Is(err, ErrJobNotRunning) {
		t.Fatalf("publication of an interrupted job = %v, want ErrJobNotRunning", err)
	}
	count := countRows(t, db, `SELECT COUNT(*) FROM pack_versions WHERE pack_id = ?`, packID)
	if count != 0 {
		t.Fatalf("%d versions committed for an interrupted job", count)
	}
	pack, err := service.Get(ctx, packID)
	if err != nil {
		t.Fatalf("get pack: %v", err)
	}
	if pack.Status != StatusDraft || pack.CurrentVersion != 0 {
		t.Fatalf("pack is %s v%d after a rejected publication", pack.Status, pack.CurrentVersion)
	}
}

func TestCommitProcessedRejectsRevokedPack(t *testing.T) {
	service, db := openTestService(t)
	ctx := context.Background()
	packID, jobID := runningJob(t, db, service, "revoked-pack", 1)
	if _, err := db.Exec(`UPDATE packs SET status = 'revoked' WHERE id = ?`, packID); err != nil {
		t.Fatalf("revoke pack: %v", err)
	}
	result := processedVersion(packID, jobID, 1, 2)
	if err := CommitProcessed(ctx, db, result); !errors.Is(err, ErrRevoked) {
		t.Fatalf("publication into a revoked pack = %v, want ErrRevoked", err)
	}
	if count := countRows(t, db, `SELECT COUNT(*) FROM pack_versions WHERE pack_id = ?`, packID); count != 0 {
		t.Fatalf("%d versions committed into a revoked pack", count)
	}
	if count := countRows(t, db, `SELECT COUNT(*) FROM stickers WHERE pack_id = ?`, packID); count != 0 {
		t.Fatalf("%d sticker rows committed into a revoked pack", count)
	}
	if count := countRows(t, db, `SELECT COUNT(*) FROM processing_jobs WHERE id = ? AND status = 'running'`, jobID); count != 1 {
		t.Fatal("rejected publication changed the job state")
	}
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	return count
}
