package packs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/google/uuid"
)

func openTestService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(db), db
}

func testChecksum(seed string) string {
	sum := strings.Repeat(seed, 64)
	return sum[:64]
}

// addVersion inserts a committed version through the schema contract T6 owns,
// so catalog tests exercise immutable rows without depending on publication.
func addVersion(t *testing.T, db *sql.DB, packID string, version int) string {
	t.Helper()
	checksum := testChecksum("a")
	manifest, err := json.Marshal(map[string]any{
		"schema":          "restick.pack.v1",
		"id":              packID,
		"version":         version,
		"checksum_sha256": checksum,
		"stickers":        []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	outputDir := fmt.Sprintf("packs/%s/v%d", packID, version)
	if _, err := db.Exec(`INSERT INTO pack_versions
		(pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, ?, ?, ?, 2, ?, ?, 1024, ?, ?)`,
		packID, version, "job-"+uuid.NewString(), checksum, string(manifest),
		fmt.Sprintf("bundles/%s/v%d.zip", packID, version), outputDir+"/thumb.webp", outputDir); err != nil {
		t.Fatalf("insert version: %v", err)
	}
	for i := range 2 {
		if _, err := db.Exec(`INSERT INTO stickers
			(pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height, emoji_hint, keywords_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, 512, 512, ?, '[]')`,
			packID, version, i, fmt.Sprintf("%03d.webp", i),
			fmt.Sprintf("%s/%03d.webp", outputDir, i),
			fmt.Sprintf("%s/thumbs/thumb_%03d.webp", outputDir, i),
			testChecksum("b"), ""); err != nil {
			t.Fatalf("insert sticker: %v", err)
		}
	}
	if _, err := db.Exec(`UPDATE packs SET status = 'ready', current_version = ? WHERE id = ?`, version, packID); err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	return checksum
}

func insertQueuedJob(t *testing.T, db *sql.DB, packID string, version int) string {
	t.Helper()
	id := "job-" + uuid.NewString()
	if _, err := db.Exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json)
		VALUES (?, ?, ?, 'queued', '{"schema_version":1}')`, id, packID, version); err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func TestCatalogEditDoesNotMutateCommittedManifest(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	pack, err := svc.Create(ctx, PackDraft{
		Slug:        "editable-pack",
		Name:        "Original",
		Author:      "Author",
		Description: "original description",
		Tags:        []string{"cute", "animals"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	checksum := addVersion(t, db, pack.ID, 1)
	before, err := svc.GetVersion(ctx, pack.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion before: %v", err)
	}

	updated, err := svc.UpdateCatalog(ctx, pack.ID, Metadata{
		Name:        "Renamed",
		Author:      "New Author",
		Description: "new description",
		Tags:        []string{"sale"},
	})
	if err != nil {
		t.Fatalf("UpdateCatalog: %v", err)
	}
	if updated.Name != "Renamed" || updated.Author != "New Author" || updated.Description != "new description" {
		t.Errorf("catalog metadata = %+v", updated)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "sale" {
		t.Errorf("tags = %v", updated.Tags)
	}
	if updated.Status != StatusReady || updated.CurrentVersion != 1 || updated.ChecksumSHA256 != checksum {
		t.Errorf("catalog summary changed unexpectedly: %+v", updated)
	}

	after, err := svc.GetVersion(ctx, pack.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion after: %v", err)
	}
	if string(after.ManifestJSON) != string(before.ManifestJSON) {
		t.Errorf("manifest_json mutated: %s -> %s", before.ManifestJSON, after.ManifestJSON)
	}
	if after.ChecksumSHA256 != before.ChecksumSHA256 || after.BundlePath != before.BundlePath ||
		after.ThumbnailPath != before.ThumbnailPath || after.StickerCount != before.StickerCount {
		t.Errorf("version metadata mutated: %+v -> %+v", before, after)
	}
	if len(after.Stickers) != 2 || after.Stickers[0].ObjectPath != before.Stickers[0].ObjectPath {
		t.Errorf("sticker rows mutated: %+v", after.Stickers)
	}

	// A direct check guards against catalog SQL touching the immutable row.
	var manifest string
	var bundlePath string
	var storedChecksum string
	if err := db.QueryRow(`SELECT manifest_json, bundle_path, checksum_sha256 FROM pack_versions WHERE pack_id = ? AND version = 1`, pack.ID).
		Scan(&manifest, &bundlePath, &storedChecksum); err != nil {
		t.Fatal(err)
	}
	if manifest != string(before.ManifestJSON) || bundlePath != before.BundlePath || storedChecksum != before.ChecksumSHA256 {
		t.Errorf("pack_versions row changed")
	}
}

func TestSlugRemainsReservedAfterRevocation(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	pack, err := svc.Create(ctx, PackDraft{Slug: "reserved-pack", Name: "Reserved", Author: "Owner"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	checksum := addVersion(t, db, pack.ID, 1)
	if _, err := db.Exec(`UPDATE packs SET status = 'revoked', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`, pack.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO pack_tombstones (pack_id, version, revoked_at, reason_code, checksum_sha256)
		VALUES (?, 1, '2026-11-10T00:00:00Z', 'owner_request', ?)`, pack.ID, checksum); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Create(ctx, PackDraft{Slug: "reserved-pack", Name: "Copy", Author: "Owner"}); !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("duplicate slug error = %v, want ErrSlugTaken", err)
	}
	revoked, err := svc.Get(ctx, pack.ID)
	if err != nil {
		t.Fatalf("Get revoked: %v", err)
	}
	if revoked.Status != StatusRevoked {
		t.Errorf("status = %q", revoked.Status)
	}
	if _, err := svc.UpdateCatalog(ctx, pack.ID, Metadata{Name: "Nope", Author: "Owner"}); !errors.Is(err, ErrRevoked) {
		t.Errorf("UpdateCatalog revoked error = %v, want ErrRevoked", err)
	}
	if _, err := svc.Create(ctx, PackDraft{Slug: "fresh-pack", Name: "Fresh", Author: "Owner"}); err != nil {
		t.Errorf("new slug after revoke: %v", err)
	}
	if err := svc.DeleteDraft(ctx, pack.ID); !errors.Is(err, ErrNotDraft) {
		t.Errorf("DeleteDraft revoked error = %v, want ErrNotDraft", err)
	}
}

func TestDraftDeleteRejectsActiveJob(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	draft, err := svc.Create(ctx, PackDraft{Slug: "draft-pack", Name: "Draft", Author: "Owner", Tags: []string{"temp"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	jobID := insertQueuedJob(t, db, draft.ID, 1)
	if err := svc.DeleteDraft(ctx, draft.ID); !errors.Is(err, ErrJobActive) {
		t.Fatalf("DeleteDraft active error = %v, want ErrJobActive", err)
	}
	if _, err := svc.Get(ctx, draft.ID); err != nil {
		t.Fatalf("pack disappeared while job active: %v", err)
	}

	if _, err := db.Exec(`UPDATE processing_jobs SET status = 'failed' WHERE id = ?`, jobID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteDraft(ctx, draft.ID); err != nil {
		t.Fatalf("DeleteDraft after job: %v", err)
	}
	if _, err := svc.Get(ctx, draft.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted draft Get error = %v, want ErrNotFound", err)
	}
	var orphanTags int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pack_tags WHERE pack_id = ?`, draft.ID).Scan(&orphanTags); err != nil {
		t.Fatal(err)
	}
	if orphanTags != 0 {
		t.Errorf("pack_tags rows survived delete: %d", orphanTags)
	}
	if _, err := svc.Create(ctx, PackDraft{Slug: "draft-pack", Name: "Recreated", Author: "Owner"}); err != nil {
		t.Errorf("deleted draft slug was not freed: %v", err)
	}

	ready, err := svc.Create(ctx, PackDraft{Slug: "ready-pack", Name: "Ready", Author: "Owner"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addVersion(t, db, ready.ID, 1)
	if err := svc.DeleteDraft(ctx, ready.ID); !errors.Is(err, ErrNotDraft) {
		t.Fatalf("DeleteDraft ready error = %v, want ErrNotDraft", err)
	}
}

func TestSetSourceKeepsUncommittedSourceFields(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	pack, err := svc.Create(ctx, PackDraft{Slug: "source-pack", Name: "Source", Author: "Owner"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addVersion(t, db, pack.ID, 1)
	before, err := svc.GetVersion(ctx, pack.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	source := assets.Source{
		ID:             uuid.NewString(),
		Filename:       "sheet.webp",
		MediaType:      "image/webp",
		ByteSize:       2048,
		Width:          512,
		Height:         512,
		ChecksumSHA256: testChecksum("c"),
		RelativePath:   "sources/" + uuid.NewString() + "/original.webp",
	}
	updated, err := svc.SetSource(ctx, pack.ID, source)
	if err != nil {
		t.Fatalf("SetSource: %v", err)
	}
	if updated.Source == nil || updated.Source.ID != source.ID || updated.Source.ChecksumSHA256 != source.ChecksumSHA256 ||
		updated.Source.ByteSize != source.ByteSize {
		t.Errorf("source = %+v", updated.Source)
	}
	after, err := svc.GetVersion(ctx, pack.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(after.ManifestJSON) != string(before.ManifestJSON) {
		t.Errorf("source attach mutated committed manifest")
	}

	jobID := insertQueuedJob(t, db, pack.ID, 2)
	if _, err := svc.SetSource(ctx, pack.ID, source); !errors.Is(err, ErrJobActive) {
		t.Errorf("SetSource with active job error = %v, want ErrJobActive", err)
	}
	if _, err := db.Exec(`UPDATE processing_jobs SET status = 'failed' WHERE id = ?`, jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE packs SET status = 'revoked' WHERE id = ?`, pack.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetSource(ctx, pack.ID, source); !errors.Is(err, ErrRevoked) {
		t.Errorf("SetSource revoked error = %v, want ErrRevoked", err)
	}
}

func TestListReadyFiltersPaginatesAndOrders(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	packs := make([]Pack, 3)
	for i := range 3 {
		created, err := svc.Create(ctx, PackDraft{
			Slug:   fmt.Sprintf("pack-%d", i+1),
			Name:   fmt.Sprintf("Pack %d", i+1),
			Author: "Owner",
			Tags:   [][]string{{"sale", "new"}, {"new"}, {"sale"}}[i],
		})
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		addVersion(t, db, created.ID, 1)
		packs[i] = created
		if _, err := db.Exec(`UPDATE packs SET updated_at = ? WHERE id = ?`,
			fmt.Sprintf("2026-01-0%dT00:00:00Z", i+1), created.ID); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := svc.Create(ctx, PackDraft{Slug: "still-draft", Name: "Draft", Author: "Owner"})
	if err != nil {
		t.Fatalf("Create draft: %v", err)
	}

	first, err := svc.ListReady(ctx, Filter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListReady page 1: %v", err)
	}
	if first.Total != 3 || first.Page != 1 || first.PageSize != 2 || len(first.Items) != 2 {
		t.Fatalf("page 1 = %+v", first)
	}
	if first.Items[0].ID != packs[2].ID || first.Items[1].ID != packs[1].ID {
		t.Errorf("page 1 order = %s, %s", first.Items[0].Slug, first.Items[1].Slug)
	}
	second, err := svc.ListReady(ctx, Filter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListReady page 2: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != packs[0].ID {
		t.Errorf("page 2 items = %+v", second.Items)
	}

	filtered, err := svc.ListReady(ctx, Filter{Tag: "sale", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListReady tag: %v", err)
	}
	if filtered.Total != 2 || len(filtered.Items) != 2 ||
		filtered.Items[0].ID != packs[2].ID || filtered.Items[1].ID != packs[0].ID {
		t.Errorf("tag filter = %+v", filtered)
	}

	clamped, err := svc.ListReady(ctx, Filter{PageSize: 1000})
	if err != nil {
		t.Fatalf("ListReady clamp: %v", err)
	}
	if clamped.Page != 1 || clamped.PageSize != 100 {
		t.Errorf("clamped page = %+v", clamped)
	}

	got, err := svc.Get(ctx, draft.ID)
	if err != nil {
		t.Fatalf("Get draft: %v", err)
	}
	if got.Status != StatusDraft {
		t.Errorf("draft status = %q", got.Status)
	}
}
