package httpx

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/storage"
)

const testPublicBase = "https://sticker.example"

type publicFixture struct {
	t       *testing.T
	root    string
	db      *sql.DB
	catalog *packs.Service
	files   *assets.Store
	handler http.Handler
	auth    *auth.Service
	apiKey  string
	keyID   string
	session string
}

type testPack struct {
	pack     packs.Pack
	checksum string
	bundle   []byte
	files    map[string][]byte
}

func newPublicFixture(t *testing.T) *publicFixture {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	db, err := storage.Open(ctx, root)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := assets.NewStore(root)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	authSvc := auth.NewService(db)
	if err := authSvc.BootstrapOwner(ctx, []byte("owner-password")); err != nil {
		t.Fatalf("BootstrapOwner: %v", err)
	}
	session, err := authSvc.Login(ctx, "owner-password", time.Now())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	key, err := authSvc.CreateKey(ctx, "sdk", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	catalog := packs.NewService(db)
	return &publicFixture{
		t: t, root: root, db: db, catalog: catalog, files: files,
		handler: NewPublicHandler(catalog, files, authSvc, testPublicBase+"/"),
		auth:    authSvc, apiKey: key.Raw, keyID: key.ID, session: session,
	}
}

// publish writes a real committed v1 layout (files, ZIP, DB rows) for a pack
// so the handlers run against actual SQLite and disk.
func (f *publicFixture) publish(slug string, stickerCount int, tags ...string) testPack {
	f.t.Helper()
	p, err := f.catalog.Create(context.Background(), seedSheetDraft(f.t, f.db, f.files, slug, "Pack "+slug, tags))
	if err != nil {
		f.t.Fatalf("packs.Create: %v", err)
	}
	return f.writeVersion(p.ID, slug, p.Name, p.Author, 1, stickerCount)
}

// publishNextVersion commits another immutable version on a ready pack.
func (f *publicFixture) publishNextVersion(existing testPack, version int) testPack {
	f.t.Helper()
	return f.writeVersion(existing.pack.ID, existing.pack.Slug, existing.pack.Name, existing.pack.Author, version, existing.pack.StickerCount)
}

func (f *publicFixture) writeVersion(packID, slug, name, author string, version, stickerCount int) testPack {
	f.t.Helper()
	ctx := context.Background()
	p := packs.Pack{ID: packID}
	outputDir := fmt.Sprintf("packs/%s/v%d", packID, version)
	dir := filepath.Join(f.root, outputDir)
	if err := os.MkdirAll(filepath.Join(dir, "thumbs"), 0o755); err != nil {
		f.t.Fatal(err)
	}

	objects := make([][]byte, stickerCount)
	thumbs := make([][]byte, stickerCount)
	diskFiles := map[string][]byte{}
	for i := range stickerCount {
		objects[i] = []byte(fmt.Sprintf("sticker-bytes-%d-%s-%d", i, slug, version))
		thumbs[i] = []byte(fmt.Sprintf("thumb-bytes-%d-%s-%d", i, slug, version))
		diskFiles[fmt.Sprintf("%s/%03d.webp", outputDir, i)] = objects[i]
		diskFiles[fmt.Sprintf("%s/thumbs/thumb_%03d.webp", outputDir, i)] = thumbs[i]
	}
	diskFiles[outputDir+"/thumb.webp"] = thumbs[0]

	manifestStickers := make([]map[string]any, 0, stickerCount)
	for i := range stickerCount {
		manifestStickers = append(manifestStickers, map[string]any{
			"file": fmt.Sprintf("%03d.webp", i), "thumb": fmt.Sprintf("thumbs/thumb_%03d.webp", i),
			"emoji_hint": "", "width": assets.StickerDimension, "height": assets.StickerDimension,
		})
	}
	joined := []byte{}
	for _, object := range objects {
		joined = append(joined, object...)
	}
	sum := sha256.Sum256(joined)
	checksum := hex.EncodeToString(sum[:])
	manifest := map[string]any{
		"schema": "restick.pack.v1", "id": p.ID, "slug": slug, "name": name, "author": author,
		"description": "", "tags": []string{}, "ip_policy": "original_only", "version": version,
		"checksum_sha256": checksum, "stickers": manifestStickers,
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		f.t.Fatal(err)
	}
	diskFiles[outputDir+"/pack.json"] = manifestJSON

	for rel, body := range diskFiles {
		if err := os.WriteFile(filepath.Join(f.root, rel), body, 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	var bundle bytes.Buffer
	zw := zip.NewWriter(&bundle)
	for rel, body := range diskFiles {
		entry, err := zw.Create(strings.TrimPrefix(rel, outputDir+"/"))
		if err != nil {
			f.t.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			f.t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		f.t.Fatal(err)
	}
	bundlePath := fmt.Sprintf("bundles/%s/v%d.zip", p.ID, version)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(f.root, bundlePath)), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, bundlePath), bundle.Bytes(), 0o644); err != nil {
		f.t.Fatal(err)
	}

	if _, err := f.db.Exec(`INSERT INTO pack_versions
		(pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, version, fmt.Sprintf("job-%s-v%d", p.ID, version), checksum, stickerCount, string(manifestJSON), bundlePath,
		len(bundle.Bytes()), outputDir+"/thumb.webp", outputDir); err != nil {
		f.t.Fatalf("insert version: %v", err)
	}
	for i := range stickerCount {
		objectSum := sha256.Sum256(objects[i])
		if _, err := f.db.Exec(`INSERT INTO stickers
			(pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height, emoji_hint, keywords_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', '["demo"]')`,
			p.ID, version, i, fmt.Sprintf("%03d.webp", i), fmt.Sprintf("%s/%03d.webp", outputDir, i),
			fmt.Sprintf("%s/thumbs/thumb_%03d.webp", outputDir, i),
			hex.EncodeToString(objectSum[:]), assets.StickerDimension, assets.StickerDimension); err != nil {
			f.t.Fatalf("insert sticker: %v", err)
		}
	}
	if _, err := f.db.Exec(`UPDATE packs SET status = 'ready', current_version = ? WHERE id = ?`, version, p.ID); err != nil {
		f.t.Fatalf("mark ready: %v", err)
	}
	ready, err := f.catalog.Get(ctx, p.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	return testPack{pack: ready, checksum: checksum, bundle: bundle.Bytes(), files: diskFiles}
}

func (f *publicFixture) revoke(id string, revokedAt string) {
	f.t.Helper()
	if _, err := f.db.Exec(`UPDATE packs SET status = 'revoked' WHERE id = ?`, id); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.db.Exec(`INSERT INTO pack_tombstones (pack_id, version, revoked_at, reason_code, checksum_sha256)
		VALUES (?, 1, ?, 'revoked', ?)`, id, revokedAt, strings.Repeat("a", 64)); err != nil {
		f.t.Fatal(err)
	}
}

func (f *publicFixture) pinUpdatedAt(id, updatedAt string) {
	f.t.Helper()
	if _, err := f.db.Exec(`UPDATE packs SET updated_at = ? WHERE id = ?`, updatedAt, id); err != nil {
		f.t.Fatal(err)
	}
}

func (f *publicFixture) request(method, target string, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

// grant adds packs to the fixture key's grants, keeping the existing ones.
func (f *publicFixture) grant(keyID string, packIDs ...string) {
	f.t.Helper()
	ctx := context.Background()
	current, err := f.auth.KeyPacks(ctx, keyID)
	if err != nil {
		f.t.Fatalf("KeyPacks: %v", err)
	}
	if err := f.auth.SetKeyPacks(ctx, keyID, append(current, packIDs...)); err != nil {
		f.t.Fatalf("SetKeyPacks: %v", err)
	}
}

func (f *publicFixture) keyHeaders() map[string]string {
	return map[string]string{"X-Api-Key": f.apiKey}
}

func (f *publicFixture) ownerHeaders() map[string]string {
	return map[string]string{"Cookie": adminCookieName + "=" + f.session}
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return payload
}

func TestPublicCatalogVisibilityAndPagination(t *testing.T) {
	f := newPublicFixture(t)
	alpha := f.publish("alpha", 2, "animals", "cute")
	second := f.publish("beta", 1, "animals")
	gamma := f.publish("gamma", 1, "space")
	draft, err := f.catalog.Create(context.Background(), seedSheetDraft(t, f.db, f.files, "draft-pack", "Draft", nil))
	if err != nil {
		t.Fatal(err)
	}
	f.grant(f.keyID, alpha.pack.ID, second.pack.ID, gamma.pack.ID, draft.ID)
	f.revoke(second.pack.ID, "2026-11-10T10:00:00Z")
	// Pin fixed UTC-second timestamps so ordering never depends on wall clock
	// or random UUIDs: gamma is newer, so it sorts first.
	f.pinUpdatedAt(alpha.pack.ID, "2026-11-10T09:00:00Z")
	f.pinUpdatedAt(gamma.pack.ID, "2026-11-10T09:00:01Z")

	unauthorized := f.request(http.MethodGet, "/api/v1/packs", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("no key status = %d", unauthorized.Code)
	}
	if got := unauthorized.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("metadata cache = %q", got)
	}
	if code := decodeBody(t, unauthorized)["error"].(map[string]any)["code"]; code != "unauthorized" {
		t.Fatalf("error code = %v", code)
	}

	rec := f.request(http.MethodGet, "/api/v1/packs", f.keyHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body)
	}
	body := decodeBody(t, rec)
	if body["page"].(float64) != 1 || body["page_size"].(float64) != 20 || body["total"].(float64) != 2 {
		t.Fatalf("page = %v/%v/%v", body["page"], body["page_size"], body["total"])
	}
	items := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %d", len(items))
	}
	ids := []string{}
	for _, item := range items {
		id := item.(map[string]any)["id"].(string)
		ids = append(ids, id)
		if id == draft.ID || id == second.pack.ID {
			t.Fatalf("hidden pack %s visible", id)
		}
	}
	if len(ids) != 2 || ids[0] != gamma.pack.ID || ids[1] != alpha.pack.ID {
		t.Fatalf("items must be ordered updated_at DESC, id ASC: %v", ids)
	}

	paged := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs?page=2&page_size=1", f.keyHeaders()))
	if paged["total"].(float64) != 2 || len(paged["items"].([]any)) != 1 {
		t.Fatalf("paged = %v", paged)
	}
	if secondPageID := paged["items"].([]any)[0].(map[string]any)["id"]; secondPageID != alpha.pack.ID {
		t.Fatalf("page 2 first id = %v, want %s", secondPageID, alpha.pack.ID)
	}
	filtered := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs?tag=space", f.keyHeaders()))
	if filtered["total"].(float64) != 1 {
		t.Fatalf("tag filter = %v", filtered["total"])
	}
}

func TestExistingSDKResponseFields(t *testing.T) {
	f := newPublicFixture(t)
	created := f.publish("alpha", 2, "cute")
	id := created.pack.ID
	f.grant(f.keyID, id)

	list := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs", f.keyHeaders()))
	item := list["items"].([]any)[0].(map[string]any)
	assertSummaryFields(t, item)
	if got := item["thumbnail_url"]; got != testPublicBase+"/v1/objects/packs/"+id+"/v1/thumb.webp" {
		t.Fatalf("thumbnail_url = %v", got)
	}

	detail := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs/"+id, f.keyHeaders()))
	assertSummaryFields(t, detail)
	stickers := detail["stickers"].([]any)
	if len(stickers) != 2 {
		t.Fatalf("stickers = %d", len(stickers))
	}
	sticker := stickers[0].(map[string]any)
	for _, field := range []string{"file", "emoji_hint", "keywords", "width", "height", "file_url", "thumb_url"} {
		if _, ok := sticker[field]; !ok {
			t.Fatalf("sticker missing %s: %v", field, sticker)
		}
	}
	if sticker["file"] != "000.webp" || sticker["width"].(float64) != 512 || sticker["height"].(float64) != 512 {
		t.Fatalf("sticker = %v", sticker)
	}
	if sticker["file_url"] != testPublicBase+"/v1/objects/packs/"+id+"/v1/000.webp" {
		t.Fatalf("file_url = %v", sticker["file_url"])
	}
	if sticker["thumb_url"] != testPublicBase+"/v1/objects/packs/"+id+"/v1/thumbs/thumb_000.webp" {
		t.Fatalf("thumb_url = %v", sticker["thumb_url"])
	}
	if keywords, ok := sticker["keywords"].([]any); !ok || len(keywords) != 1 {
		t.Fatalf("keywords = %v", sticker["keywords"])
	}

	head := f.request(http.MethodHead, "/api/v1/packs/"+id, f.keyHeaders())
	if head.Code != http.StatusOK || head.Header().Get("ETag") != `"sha256:`+created.checksum+`"` {
		t.Fatalf("HEAD status=%d etag=%q", head.Code, head.Header().Get("ETag"))
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body = %q", head.Body)
	}

	bundle := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs/"+id+"/bundle", f.keyHeaders()))
	if bundle["url"] != testPublicBase+"/v1/bundles/"+id+"/v/1" ||
		bundle["checksum_sha256"] != created.checksum || bundle["version"].(float64) != 1 {
		t.Fatalf("bundle meta = %v", bundle)
	}
}

func assertSummaryFields(t *testing.T, payload map[string]any) {
	t.Helper()
	for _, field := range []string{"id", "slug", "name", "author", "tags", "sticker_count", "version", "thumbnail_url", "checksum_sha256"} {
		if _, ok := payload[field]; !ok {
			t.Fatalf("summary missing %s: %v", field, payload)
		}
	}
	if payload["tags"] == nil {
		t.Fatal("tags must be an array, not null")
	}
}

func TestTombstoneInclusiveTimestamp(t *testing.T) {
	f := newPublicFixture(t)
	first := f.publish("alpha", 1)
	second := f.publish("beta", 1)
	old := f.publish("gamma", 1)
	f.grant(f.keyID, first.pack.ID, second.pack.ID, old.pack.ID)
	f.revoke(old.pack.ID, "2026-11-10T09:00:00Z")
	f.revoke(first.pack.ID, "2026-11-10T10:00:01Z")
	f.revoke(second.pack.ID, "2026-11-10T10:00:01Z")

	all := decodeBody(t, f.request(http.MethodGet, "/api/v1/tombstones", f.keyHeaders()))
	if total := len(all["items"].([]any)); total != 3 {
		t.Fatalf("all items = %d", total)
	}

	since := decodeBody(t, f.request(http.MethodGet, "/api/v1/tombstones?since=2026-11-10T10:00:01Z", f.keyHeaders()))
	lowerID := min(first.pack.ID, second.pack.ID)
	items := since["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("inclusive since items = %d", len(items))
	}
	for i, item := range items {
		entry := item.(map[string]any)
		if entry["revoked_at"] != "2026-11-10T10:00:01Z" {
			t.Fatalf("revoked_at = %v", entry["revoked_at"])
		}
		for _, field := range []string{"pack_id", "revoked_at", "reason_code", "checksum_sha256"} {
			if _, ok := entry[field]; !ok {
				t.Fatalf("tombstone missing %s: %v", field, entry)
			}
		}
		if i == 0 && entry["pack_id"] != lowerID {
			t.Fatalf("same-second order = %v, want pack_id order %s", entry["pack_id"], lowerID)
		}
	}

	empty := decodeBody(t, f.request(http.MethodGet, "/api/v1/tombstones?since=2026-11-10T10:00:02Z", f.keyHeaders()))
	if len(empty["items"].([]any)) != 0 {
		t.Fatalf("future since returned %v", empty["items"])
	}

	invalid := f.request(http.MethodGet, "/api/v1/tombstones?since=10-minutes-ago", f.keyHeaders())
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid since status = %d", invalid.Code)
	}
}

func (f *publicFixture) allRoutes(id string) map[string]string {
	return map[string]string{
		"detail":      "/api/v1/packs/" + id,
		"bundle meta": "/api/v1/packs/" + id + "/bundle",
		"object":      "/v1/objects/packs/" + id + "/v1/000.webp",
		"bundle":      "/v1/bundles/" + id + "/v/1",
	}
}

func (f *publicFixture) expectStatus(headers map[string]string, id string, want int) {
	f.t.Helper()
	for name, target := range f.allRoutes(id) {
		if rec := f.request(http.MethodGet, target, headers); rec.Code != want {
			f.t.Errorf("%s %s = %d, want %d", name, target, rec.Code, want)
		}
	}
}

func TestKeyWithoutGrantsSeesNothing(t *testing.T) {
	f := newPublicFixture(t)
	p := f.publish("alpha", 1)
	rec := f.request(http.MethodGet, "/api/v1/packs", f.keyHeaders())
	body := decodeBody(t, rec)
	if rec.Code != http.StatusOK || len(body["items"].([]any)) != 0 || body["total"].(float64) != 0 {
		t.Fatalf("list = %d %v", rec.Code, body)
	}
	f.expectStatus(f.keyHeaders(), p.pack.ID, http.StatusNotFound)
}

func TestGrantedPackVisibleOnAllRoutes(t *testing.T) {
	f := newPublicFixture(t)
	a := f.publish("alpha", 1)
	b := f.publish("beta", 1)
	f.grant(f.keyID, a.pack.ID)
	body := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs", f.keyHeaders()))
	items := body["items"].([]any)
	if len(items) != 1 || body["total"].(float64) != 1 || items[0].(map[string]any)["id"] != a.pack.ID {
		t.Fatalf("list = %v", body)
	}
	f.expectStatus(f.keyHeaders(), a.pack.ID, http.StatusOK)
	f.expectStatus(f.keyHeaders(), b.pack.ID, http.StatusNotFound)
	if rec := f.request(http.MethodHead, "/api/v1/packs/"+b.pack.ID, f.keyHeaders()); rec.Code != http.StatusNotFound {
		t.Fatalf("HEAD non-granted = %d", rec.Code)
	}
	if rec := f.request(http.MethodHead, "/api/v1/packs/"+a.pack.ID, f.keyHeaders()); rec.Code != http.StatusOK {
		t.Fatalf("HEAD granted = %d", rec.Code)
	}
}

func TestNonGrantedRevokedPackIs404NotGone(t *testing.T) {
	f := newPublicFixture(t)
	granted := f.publish("alpha", 1)
	other := f.publish("beta", 1)
	f.grant(f.keyID, granted.pack.ID)
	f.revoke(granted.pack.ID, "2026-11-10T10:00:00Z")
	f.revoke(other.pack.ID, "2026-11-10T10:00:00Z")
	f.expectStatus(f.keyHeaders(), granted.pack.ID, http.StatusGone)
	f.expectStatus(f.keyHeaders(), other.pack.ID, http.StatusNotFound)
}

func TestOwnerSessionIgnoresGrants(t *testing.T) {
	f := newPublicFixture(t)
	p := f.publish("alpha", 1)
	cookie := map[string]string{"Cookie": adminCookieName + "=" + f.session}
	f.expectStatus(cookie, p.pack.ID, http.StatusOK)
	body := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs", cookie))
	if body["total"].(float64) != 1 {
		t.Fatalf("owner list = %v", body)
	}
}

func TestTombstonesRespectGrants(t *testing.T) {
	f := newPublicFixture(t)
	a := f.publish("alpha", 1)
	b := f.publish("beta", 1)
	f.grant(f.keyID, a.pack.ID)
	f.revoke(a.pack.ID, "2026-11-10T10:00:00Z")
	f.revoke(b.pack.ID, "2026-11-10T10:00:00Z")

	list := func(since string) []map[string]any {
		target := "/api/v1/tombstones"
		if since != "" {
			target += "?since=" + since
		}
		var out []map[string]any
		for _, item := range decodeBody(t, f.request(http.MethodGet, target, f.keyHeaders()))["items"].([]any) {
			out = append(out, item.(map[string]any))
		}
		return out
	}
	if got := list(""); len(got) != 1 || got[0]["pack_id"] != a.pack.ID || got[0]["reason_code"] != "revoked" {
		t.Fatalf("granted-only tombstones = %v", got)
	}

	c := f.publish("gamma", 1)
	f.grant(f.keyID, c.pack.ID)
	if err := f.auth.SetKeyPacks(context.Background(), f.keyID, []string{a.pack.ID}); err != nil {
		t.Fatal(err)
	}
	got := list("")
	var removal map[string]any
	for _, item := range got {
		if item["pack_id"] == c.pack.ID {
			removal = item
		}
	}
	if len(got) != 2 || removal == nil || removal["reason_code"] != "access_removed" || removal["checksum_sha256"] != "" {
		t.Fatalf("removal tombstones = %v", got)
	}
	revokedAt, err := time.Parse("2006-01-02T15:04:05Z", removal["revoked_at"].(string))
	if err != nil {
		t.Fatalf("removal revoked_at = %v", removal["revoked_at"])
	}
	// since is inclusive on the removal source and filters the revoke source.
	if n := len(list(revokedAt.Format(time.RFC3339))); n != 1 {
		t.Fatalf("since=removed_at returned %d, want only the removal", n)
	}
	if n := len(list("2026-11-10T10:00:00Z")); n != 2 {
		t.Fatalf("since=revoke time returned %d, want 2", n)
	}
	f.grant(f.keyID, c.pack.ID)
	for _, item := range list("") {
		if item["pack_id"] == c.pack.ID {
			t.Fatalf("re-granted pack still tombstoned: %v", item)
		}
	}
	// Owner sees the unfiltered revoke list, never removals.
	owner := decodeBody(t, f.request(http.MethodGet, "/api/v1/tombstones", map[string]string{"Cookie": adminCookieName + "=" + f.session}))
	if n := len(owner["items"].([]any)); n != 2 {
		t.Fatalf("owner tombstones = %d, want 2", n)
	}
}

func TestPublicListPacksIncludesPreviewsAndThemeColor(t *testing.T) {
	f := newPublicFixture(t)
	tinted := f.publish("tinted", 8, "cute")
	plain := f.publish("plain", 1)
	if _, err := f.db.Exec(`UPDATE packs SET theme_color_suggested = '#AABBCC' WHERE id = ?`, tinted.pack.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE packs SET theme_color = '#112233' WHERE id = ?`, tinted.pack.ID); err != nil {
		t.Fatal(err)
	}

	list := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs", f.ownerHeaders()))
	byID := map[string]map[string]any{}
	for _, it := range list["items"].([]any) {
		m := it.(map[string]any)
		byID[m["id"].(string)] = m
	}
	got := byID[tinted.pack.ID]
	if got["theme_color"] != "#112233" {
		t.Fatalf("theme_color = %v, want admin override", got["theme_color"])
	}
	previews := got["preview_stickers"].([]any)
	if len(previews) != 6 {
		t.Fatalf("previews = %d, want 6", len(previews))
	}
	first := previews[0].(map[string]any)
	if first["file_url"] != testPublicBase+"/v1/objects/packs/"+tinted.pack.ID+"/v1/000.webp" ||
		first["thumb_url"] != testPublicBase+"/v1/objects/packs/"+tinted.pack.ID+"/v1/thumbs/thumb_000.webp" {
		t.Fatalf("first preview = %v", first)
	}
	other := byID[plain.pack.ID]
	if other["theme_color"] != "" {
		t.Fatalf("plain theme_color = %v, want empty", other["theme_color"])
	}
	if arr, ok := other["preview_stickers"].([]any); !ok || len(arr) != 1 {
		t.Fatalf("plain previews = %v", other["preview_stickers"])
	}
}

func TestPublicEmptyPreviewsAreArrayNotNull(t *testing.T) {
	f := newPublicFixture(t)
	empty := f.publish("empty", 1)
	if _, err := f.db.Exec(`DELETE FROM stickers WHERE pack_id = ?`, empty.pack.ID); err != nil {
		t.Fatal(err)
	}
	got := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs/"+empty.pack.ID, f.ownerHeaders()))
	arr, ok := got["preview_stickers"].([]any)
	if !ok || len(arr) != 0 {
		t.Fatalf("preview_stickers = %v, want []", got["preview_stickers"])
	}
}

func TestPublicPackKeepsFilterByTagAndKey(t *testing.T) {
	f := newPublicFixture(t)
	a := f.publish("alpha", 3, "cute")
	b := f.publish("beta", 3, "other")
	f.grant(f.keyID, a.pack.ID)
	_ = b

	list := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs?tag=cute", f.keyHeaders()))
	items := list["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if previews := items[0].(map[string]any)["preview_stickers"].([]any); len(previews) != 3 {
		t.Fatalf("previews under tag+key filter = %d, want 3", len(previews))
	}
}

func TestPublicGetPackIncludesPreviews(t *testing.T) {
	f := newPublicFixture(t)
	p := f.publish("detail", 8)
	f.grant(f.keyID, p.pack.ID)
	got := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs/"+p.pack.ID, f.keyHeaders()))
	if previews := got["preview_stickers"].([]any); len(previews) != 6 {
		t.Fatalf("previews = %d, want 6", len(previews))
	}
	if got["theme_color"] != "" {
		t.Fatalf("theme_color = %v", got["theme_color"])
	}
	if len(got["stickers"].([]any)) != 8 {
		t.Fatal("full stickers list must stay intact")
	}
}
