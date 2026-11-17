package packs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

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

// insertSheet adds a design plus one sheet for it and returns both ids.
func insertSheet(t *testing.T, db *sql.DB, provider string) (string, string) {
	t.Helper()
	designID := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO designs (id, name, rows, columns, outline_width, outline_color, draft_json, snapshot_json)
		VALUES (?, 'Design', 2, 2, 4, '#FFFFFF', '{}', '{"schema_version":1}')`, designID); err != nil {
		t.Fatalf("insert design: %v", err)
	}
	return designID, insertSheetFor(t, db, designID, provider)
}

func insertSheetFor(t *testing.T, db *sql.DB, designID, provider string) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO design_sheets
		(id, design_id, provider, prompt_snapshot, filename, media_type, byte_size, width, height, checksum_sha256)
		VALUES (?, ?, ?, 'prompt', 'sheet.webp', 'image/webp', 2048, 512, 512, ?)`,
		id, designID, provider, strings.ReplaceAll(id, "-", "")[:32]+strings.ReplaceAll(id, "-", "")[:32]); err != nil {
		t.Fatalf("insert sheet: %v", err)
	}
	return id
}

// sheeted points a draft at a fresh design and sheet.
func sheeted(t *testing.T, db *sql.DB, draft PackDraft) PackDraft {
	t.Helper()
	draft.DesignID, draft.SheetID = insertSheet(t, db, "gemini")
	return draft
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

	pack, err := svc.Create(ctx, sheeted(t, db, PackDraft{
		Slug:        "editable-pack",
		Name:        "Original",
		Author:      "Author",
		Description: "original description",
		Tags:        []string{"cute", "animals"},
	}))
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
		DesignID:    pack.DesignID,
		SheetID:     pack.SheetID,
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

	pack, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "reserved-pack", Name: "Reserved", Author: "Owner"}))
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

	if _, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "reserved-pack", Name: "Copy", Author: "Owner"})); !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("duplicate slug error = %v, want ErrSlugTaken", err)
	}
	revoked, err := svc.Get(ctx, pack.ID)
	if err != nil {
		t.Fatalf("Get revoked: %v", err)
	}
	if revoked.Status != StatusRevoked {
		t.Errorf("status = %q", revoked.Status)
	}
	if _, err := svc.UpdateCatalog(ctx, pack.ID, Metadata{Name: "Nope", Author: "Owner", DesignID: pack.DesignID, SheetID: pack.SheetID}); !errors.Is(err, ErrRevoked) {
		t.Errorf("UpdateCatalog revoked error = %v, want ErrRevoked", err)
	}
	if _, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "fresh-pack", Name: "Fresh", Author: "Owner"})); err != nil {
		t.Errorf("new slug after revoke: %v", err)
	}
	if err := svc.DeleteDraft(ctx, pack.ID); !errors.Is(err, ErrNotDraft) {
		t.Errorf("DeleteDraft revoked error = %v, want ErrNotDraft", err)
	}
}

func TestDraftDeleteRejectsActiveJob(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	draft, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "draft-pack", Name: "Draft", Author: "Owner", Tags: []string{"temp"}}))
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
	if _, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "draft-pack", Name: "Recreated", Author: "Owner"})); err != nil {
		t.Errorf("deleted draft slug was not freed: %v", err)
	}

	ready, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "ready-pack", Name: "Ready", Author: "Owner"}))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addVersion(t, db, ready.ID, 1)
	if err := svc.DeleteDraft(ctx, ready.ID); !errors.Is(err, ErrNotDraft) {
		t.Fatalf("DeleteDraft ready error = %v, want ErrNotDraft", err)
	}
}

func TestCreateRequiresSheetOfSameDesign(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()
	designA, sheetA := insertSheet(t, db, "gemini")
	designB, sheetB := insertSheet(t, db, "chatgpt")

	for name, draft := range map[string]PackDraft{
		"no sheet":     {DesignID: designA},
		"no design":    {SheetID: sheetA},
		"unknown":      {DesignID: designA, SheetID: uuid.NewString()},
		"other design": {DesignID: designA, SheetID: sheetB},
	} {
		draft.Slug, draft.Name, draft.Author = "bad-"+strings.ReplaceAll(name, " ", "-"), "Bad", "Owner"
		if _, err := svc.Create(ctx, draft); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error = %v, want ErrInvalid", name, err)
		}
	}
	_ = designB

	pack, err := svc.Create(ctx, PackDraft{Slug: "good", Name: "Good", Author: "Owner", DesignID: designA, SheetID: sheetA})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pack.SheetID != sheetA || pack.Sheet == nil || pack.Sheet.Provider != "gemini" ||
		pack.Sheet.ID != sheetA || pack.Sheet.Width != 512 || pack.Sheet.ByteSize != 2048 {
		t.Errorf("pack sheet = %q %+v", pack.SheetID, pack.Sheet)
	}
}

func TestUpdateCatalogChangesSheet(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()
	design, first := insertSheet(t, db, "gemini")
	second := insertSheetFor(t, db, design, "chatgpt")

	pack, err := svc.Create(ctx, PackDraft{Slug: "swap", Name: "Swap", Author: "Owner", DesignID: design, SheetID: first})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	meta := Metadata{Name: "Swap", Author: "Owner", DesignID: design, SheetID: second}
	updated, err := svc.UpdateCatalog(ctx, pack.ID, meta)
	if err != nil {
		t.Fatalf("UpdateCatalog: %v", err)
	}
	if updated.SheetID != second || updated.Sheet == nil || updated.Sheet.Provider != "chatgpt" {
		t.Errorf("sheet = %q %+v", updated.SheetID, updated.Sheet)
	}

	// Same sheet while a job is active is a plain metadata edit.
	jobID := insertQueuedJob(t, db, pack.ID, 1)
	same := meta
	same.Name = "Renamed"
	if _, err := svc.UpdateCatalog(ctx, pack.ID, same); err != nil {
		t.Errorf("same-sheet edit with active job: %v", err)
	}
	meta.SheetID = first
	if _, err := svc.UpdateCatalog(ctx, pack.ID, meta); !errors.Is(err, ErrJobActive) {
		t.Errorf("sheet change with active job error = %v, want ErrJobActive", err)
	}
	if _, err := db.Exec(`UPDATE processing_jobs SET status = 'failed' WHERE id = ?`, jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE packs SET status = 'revoked' WHERE id = ?`, pack.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateCatalog(ctx, pack.ID, meta); !errors.Is(err, ErrRevoked) {
		t.Errorf("revoked error = %v, want ErrRevoked", err)
	}
}

func TestListReadyFiltersPaginatesAndOrders(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()

	packs := make([]Pack, 3)
	for i := range 3 {
		created, err := svc.Create(ctx, sheeted(t, db, PackDraft{
			Slug:   fmt.Sprintf("pack-%d", i+1),
			Name:   fmt.Sprintf("Pack %d", i+1),
			Author: "Owner",
			Tags:   [][]string{{"sale", "new"}, {"new"}, {"sale"}}[i],
		}))
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
	draft, err := svc.Create(ctx, sheeted(t, db, PackDraft{Slug: "still-draft", Name: "Draft", Author: "Owner"}))
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

func TestUpdateCatalogThemeColor(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()
	design, sheet := insertSheet(t, db, "gemini")
	pack, err := svc.Create(ctx, PackDraft{Slug: "tint", Name: "Tint", Author: "Owner", DesignID: design, SheetID: sheet})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	meta := func(color *string) Metadata {
		return Metadata{Name: "Tint", Author: "Owner", DesignID: design, SheetID: sheet, ThemeColor: color}
	}
	ptr := func(s string) *string { return &s }

	if _, err := svc.UpdateCatalog(ctx, pack.ID, meta(ptr("#ffdde"))); !errors.Is(err, ErrInvalid) {
		t.Fatalf("short hex error = %v, want ErrInvalid", err)
	}
	got, err := svc.UpdateCatalog(ctx, pack.ID, meta(ptr(" #ffddee ")))
	if err != nil || got.ThemeColor != "#FFDDEE" {
		t.Fatalf("set colour = %q, %v; want #FFDDEE", got.ThemeColor, err)
	}
	got, err = svc.UpdateCatalog(ctx, pack.ID, meta(nil))
	if err != nil || got.ThemeColor != "#FFDDEE" {
		t.Fatalf("nil colour = %q, %v; want kept #FFDDEE", got.ThemeColor, err)
	}
	got, err = svc.UpdateCatalog(ctx, pack.ID, meta(ptr("")))
	if err != nil || got.ThemeColor != "" {
		t.Fatalf("empty colour = %q, %v; want cleared", got.ThemeColor, err)
	}
}

func TestEffectiveThemeColor(t *testing.T) {
	for _, tc := range []struct{ color, suggested, want string }{
		{"#111111", "#222222", "#111111"},
		{"", "#222222", "#222222"},
		{"", "", ""},
	} {
		if got := (Pack{ThemeColor: tc.color, ThemeColorSuggested: tc.suggested}).EffectiveThemeColor(); got != tc.want {
			t.Errorf("Effective(%q,%q) = %q, want %q", tc.color, tc.suggested, got, tc.want)
		}
	}
}

// addStickers appends stickers sort_order from..to-1 to a committed version.
func addStickers(t *testing.T, db *sql.DB, packID string, version, from, to int) {
	t.Helper()
	dir := fmt.Sprintf("packs/%s/v%d", packID, version)
	for i := from; i < to; i++ {
		if _, err := db.Exec(`INSERT INTO stickers
			(pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height, emoji_hint, keywords_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, 512, 512, '', '[]')`,
			packID, version, i, fmt.Sprintf("%03d.webp", i), fmt.Sprintf("%s/%03d.webp", dir, i),
			fmt.Sprintf("%s/thumbs/thumb_%03d.webp", dir, i), testChecksum("b")); err != nil {
			t.Fatalf("insert sticker: %v", err)
		}
	}
}

func readyPackWithStickers(t *testing.T, svc *Service, db *sql.DB, slug string, stickers int) Pack {
	t.Helper()
	created, err := svc.Create(context.Background(), sheeted(t, db, PackDraft{Slug: slug, Name: slug, Author: "Owner"}))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	addVersion(t, db, created.ID, 1)
	addStickers(t, db, created.ID, 1, 2, stickers)
	return created
}

func TestListReadyLoadsAtMostSixPreviewsInOrder(t *testing.T) {
	svc, db := openTestService(t)
	ctx := context.Background()
	pack := readyPackWithStickers(t, svc, db, "eight", 8)

	page, err := svc.ListReady(ctx, Filter{})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("ListReady = %v, %v", page.Items, err)
	}
	dir := fmt.Sprintf("packs/%s/v1", pack.ID)
	got := page.Items[0].Previews
	if len(got) != 6 {
		t.Fatalf("previews = %d, want 6", len(got))
	}
	for i, p := range got {
		if p.ObjectPath != fmt.Sprintf("%s/%03d.webp", dir, i) || p.ThumbPath != fmt.Sprintf("%s/thumbs/thumb_%03d.webp", dir, i) {
			t.Fatalf("preview %d = %+v", i, p)
		}
	}

	// A newer version replaces the previews; v1 stickers must not leak in.
	addVersion(t, db, pack.ID, 2)
	one, err := svc.Get(ctx, pack.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Previews) != 2 || !strings.Contains(one.Previews[0].ObjectPath, "/v2/") || !strings.Contains(one.Previews[1].ObjectPath, "/v2/") {
		t.Fatalf("Get previews = %+v, want the 2 v2 stickers", one.Previews)
	}
}

// countingDriver counts queries that read the stickers table.
type countingDriver struct {
	inner driver.Driver
	n     *atomic.Int64
}

func (d countingDriver) Open(name string) (driver.Conn, error) {
	c, err := d.inner.Open(name)
	return &countingConn{Conn: c, n: d.n}, err
}

type countingConn struct {
	driver.Conn
	n *atomic.Int64
}

func (c *countingConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, "FROM stickers") {
		c.n.Add(1)
	}
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func TestListReadyPreviewsDoNotMultiplyQueries(t *testing.T) {
	dir := t.TempDir()
	setup, err := storage.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Close()
	svc := NewService(setup)
	for _, slug := range []string{"one", "two", "three"} {
		readyPackWithStickers(t, svc, setup, slug, 3)
	}

	var n atomic.Int64
	probe, err := sql.Open("sqlite", "")
	if err != nil {
		t.Fatal(err)
	}
	inner := probe.Driver()
	probe.Close()
	counted := sql.OpenDB(connectorFunc(func() (driver.Conn, error) {
		return countingDriver{inner: inner, n: &n}.Open("file:" + filepath.Join(dir, storage.DBFileName))
	}))
	defer counted.Close()

	items := []Pack{}
	page, err := svc.ListReady(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	items = append(items, page.Items...)
	if len(items) != 3 {
		t.Fatalf("items = %d", len(items))
	}
	if err := loadPreviews(context.Background(), counted, items); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 1 {
		t.Fatalf("sticker queries for 3 packs = %d, want 1", n.Load())
	}
	for _, it := range items {
		if len(it.Previews) != 3 {
			t.Fatalf("%s previews = %d", it.Slug, len(it.Previews))
		}
	}
}

type connectorFunc func() (driver.Conn, error)

func (f connectorFunc) Connect(context.Context) (driver.Conn, error) { return f() }
func (f connectorFunc) Driver() driver.Driver                        { return nil }
