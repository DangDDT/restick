package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	httpx "github.com/eykh/restick/api_svc/internal/interfaces/http"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/eykh/restick/api_svc/internal/studio"
)

const (
	ownerPassword = "acceptance-owner-password"
	jobDeadline   = 3 * time.Minute
	pollInterval  = 100 * time.Millisecond
)

// TestFactoryRoundTrip drives the production services end to end over real TLS:
// owner session, custom library fragment and frozen prompt snapshot, two real
// Python-processed versions, the SDK read surface with a client key, an
// immutable bundle under a catalog edit, revoke of every version URL, and the
// inclusive tombstone cursor. Only the network boundary is a test server; no
// service, processor or file store is faked.
func TestFactoryRoundTrip(t *testing.T) {
	env := newFactoryEnv(t)
	env.login()

	// Custom fragment followed by a compiled design snapshot.
	var group studio.Group
	env.mustAdmin(http.MethodPost, "/api/v1/admin/fragment-groups",
		studio.GroupDraft{Name: "Acceptance props", Description: "owned fixture", SortOrder: 1}, &group, http.StatusCreated)
	const originalPrompt = "a small red scarf pinned at the neck"
	var fragment studio.Fragment
	env.mustAdmin(http.MethodPost, "/api/v1/admin/fragments", studio.FragmentDraft{
		GroupID: group.ID, Name: "Khăn đỏ", NameEn: "Red scarf", Prompt: originalPrompt,
		Scope: studio.ScopeIdentity,
	}, &fragment, http.StatusCreated)

	draft := studio.Draft{
		Name: "Acceptance sheet", Rows: 2, Columns: 2,
		OutlineWidth: studio.DefaultOutlineWidth, OutlineColor: "#FFFFFF",
		FragmentIDs: []string{fragment.ID},
		Cells: []studio.Cell{
			{Action: "wave"}, {Action: "jump"}, {Action: "sleep"}, {Action: "cheer"},
		},
	}
	var compiled struct {
		CompiledPrompt string          `json:"compiled_prompt"`
		Snapshot       studio.Snapshot `json:"snapshot"`
	}
	env.mustAdmin(http.MethodPost, "/api/v1/admin/designs/compile", draft, &compiled, http.StatusOK)
	if !strings.Contains(compiled.CompiledPrompt, originalPrompt) {
		t.Fatalf("compiled prompt is missing the custom fragment:\n%s", compiled.CompiledPrompt)
	}
	var design studio.Design
	env.mustAdmin(http.MethodPost, "/api/v1/admin/designs", draft, &design, http.StatusCreated)
	if design.Prompt != compiled.CompiledPrompt {
		t.Fatalf("saved prompt differs from the compiled prompt")
	}

	// Editing the library after saving must not rewrite the stored snapshot.
	env.mustAdmin(http.MethodPut, "/api/v1/admin/fragments/"+fragment.ID, studio.FragmentDraft{
		GroupID: group.ID, Name: "Khăn đỏ", NameEn: "Red scarf",
		Prompt: "a blue scarf pinned at the neck", Scope: studio.ScopeIdentity,
	}, nil, http.StatusOK)
	var reloaded studio.Design
	env.mustAdmin(http.MethodGet, "/api/v1/admin/designs/"+design.ID, nil, &reloaded, http.StatusOK)
	if reloaded.Prompt != design.Prompt {
		t.Fatalf("design prompt changed after editing the library fragment")
	}
	if len(reloaded.Snapshot.Selections) != 1 || reloaded.Snapshot.Selections[0].Prompt != originalPrompt {
		t.Fatalf("design snapshot changed after editing the library fragment: %+v", reloaded.Snapshot.Selections)
	}

	var pack packView
	env.mustAdmin(http.MethodPost, "/api/v1/admin/packs", map[string]any{
		"slug": "acceptance-pack", "name": "Acceptance Pack", "author": "Owner",
		"description": "round trip", "tags": []string{"acceptance"}, "design_id": design.ID,
	}, &pack, http.StatusCreated)

	// Version 1: upload the owned sheet, run the real Python child, publish.
	env.uploadSource(pack.ID, sheetPNG(quadrantColorsA))
	job := env.process(pack.ID)
	if job.Version != 1 {
		t.Fatalf("first job version = %d, want 1", job.Version)
	}
	v1 := env.captureVersion(pack.ID, "acceptance")

	// Version 2: a different sheet proves the current pointer and checksum move.
	env.uploadSource(pack.ID, sheetPNG(quadrantColorsB))
	job = env.process(pack.ID)
	if job.Version != 2 {
		t.Fatalf("second job version = %d, want 2", job.Version)
	}
	v2 := env.captureVersion(pack.ID, "acceptance")
	if v1.checksum == v2.checksum {
		t.Fatalf("both versions have the same checksum %s", v2.checksum)
	}
	// Retained committed versions stay downloadable while the pack is ready.
	status, _, oldZip := env.keyCall(http.MethodGet, v1.bundleURL)
	if status != http.StatusOK || !bytes.Equal(oldZip, v1.zipBytes) {
		t.Fatalf("old version bundle while ready: status %d, bytes equal %v", status, bytes.Equal(oldZip, v1.zipBytes))
	}

	// Catalog metadata edits leave the committed ZIP bytes and checksum alone.
	env.mustAdmin(http.MethodPut, "/api/v1/admin/packs/"+pack.ID, map[string]any{
		"name": "Acceptance Pack Renamed", "author": "Owner", "description": "catalog edit",
		"tags": []string{"acceptance", "edited"}, "design_id": design.ID,
	}, nil, http.StatusOK)
	var edited sdkPack
	status, _, data := env.keyCall(http.MethodGet, env.baseURL+"/api/v1/packs/"+pack.ID)
	env.expectStatus("SDK pack detail after edit", status, http.StatusOK, data)
	if err := json.Unmarshal(data, &edited); err != nil {
		t.Fatalf("decode edited pack: %v", err)
	}
	if edited.Name != "Acceptance Pack Renamed" || edited.ChecksumSHA256 != v2.checksum || edited.Version != 2 {
		t.Fatalf("catalog edit changed version data: %+v", edited)
	}
	status, _, afterEdit := env.keyCall(http.MethodGet, v2.bundleURL)
	if status != http.StatusOK || !bytes.Equal(afterEdit, v2.zipBytes) {
		t.Fatalf("catalog edit changed ZIP bytes: status %d, bytes equal %v", status, bytes.Equal(afterEdit, v2.zipBytes))
	}

	// Revoke blocks every captured object, thumbnail and bundle URL of both versions.
	env.mustAdmin(http.MethodPost, "/api/v1/admin/packs/"+pack.ID+"/revoke", nil, nil, http.StatusOK)
	for _, url := range v1.allURLs() {
		env.expectGone(url)
	}
	for _, url := range v2.allURLs() {
		env.expectGone(url)
	}

	// The tombstone carries the current version checksum and an inclusive cursor.
	tombstones := env.tombstones("2020-01-01T00:00:00Z")
	if len(tombstones.Items) != 1 {
		t.Fatalf("tombstone count = %d, want 1", len(tombstones.Items))
	}
	tombstone := tombstones.Items[0]
	if tombstone.PackID != pack.ID || tombstone.ChecksumSHA256 != v2.checksum ||
		tombstone.ReasonCode != "revoked" || len(tombstone.RevokedAt) != len("2006-01-02T15:04:05Z") {
		t.Fatalf("unexpected tombstone: %+v", tombstone)
	}

	// Two revokes inside one UTC second would hide a timestamp rewrite, so pin a
	// fixed older persisted timestamp before the repeated revoke.
	const fixedCursor = "2026-01-02T03:04:05Z"
	if _, err := env.db.ExecContext(env.ctx, "UPDATE pack_tombstones SET revoked_at = ? WHERE pack_id = ?", fixedCursor, pack.ID); err != nil {
		t.Fatalf("pin tombstone timestamp: %v", err)
	}
	env.mustAdmin(http.MethodPost, "/api/v1/admin/packs/"+pack.ID+"/revoke", nil, nil, http.StatusOK)
	var count int
	var stamp string
	if err := env.db.QueryRowContext(env.ctx,
		"SELECT COUNT(*), MIN(revoked_at) FROM pack_tombstones WHERE pack_id = ?", pack.ID).Scan(&count, &stamp); err != nil {
		t.Fatalf("read tombstone after repeat revoke: %v", err)
	}
	if count != 1 || stamp != fixedCursor {
		t.Fatalf("repeat revoke changed the tombstone: count=%d stamp=%q", count, stamp)
	}
	if got := env.tombstones(fixedCursor); len(got.Items) != 1 || got.Items[0].RevokedAt != fixedCursor {
		t.Fatalf("inclusive since cursor missed the tombstone: %+v", got.Items)
	}
	if got := env.tombstones("2026-01-02T03:04:06Z"); len(got.Items) != 0 {
		t.Fatalf("insertion-order cursor returned later tombstones: %+v", got.Items)
	}

	// A read key cannot reach any admin write.
	payload := []byte(`{"name":"forbidden-key-write","description":"","sort_order":9}`)
	status, _, data = env.call(env.key, http.MethodPost, env.baseURL+"/api/v1/admin/fragment-groups", map[string]string{
		"X-Api-Key": env.apiKey, "Content-Type": "application/json", "Origin": env.baseURL,
	}, payload)
	if status != http.StatusUnauthorized {
		t.Fatalf("client key admin write status = %d, want 401: %s", status, data)
	}
	var groups struct {
		Items []studio.Group `json:"items"`
	}
	env.mustAdmin(http.MethodGet, "/api/v1/admin/fragment-groups", nil, &groups, http.StatusOK)
	for _, item := range groups.Items {
		if item.Name == "forbidden-key-write" {
			t.Fatalf("client key created admin data")
		}
	}
}

// factoryEnv owns one production service graph on a real data directory.
type factoryEnv struct {
	t       *testing.T
	ctx     context.Context
	db      *sql.DB
	baseURL string
	admin   *http.Client
	key     *http.Client
	apiKey  string
}

func newFactoryEnv(t *testing.T) *factoryEnv {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	db, err := storage.Open(ctx, root)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	files, err := assets.NewStore(root)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	authSvc := auth.NewService(db)
	if err := authSvc.BootstrapOwner(ctx, []byte(ownerPassword)); err != nil {
		t.Fatalf("BootstrapOwner: %v", err)
	}
	studioSvc := studio.NewService(db, files)
	catalog := packs.NewService(db)
	runner := jobs.NewRunner(db, files, jobs.PythonProcessor{}, 5*time.Minute)

	// The TLS origin is known from the listener, so the reviewed handler can be
	// built with the exact public base URL before the server starts accepting.
	server := httptest.NewUnstartedServer(nil)
	baseURL := "https://" + server.Listener.Addr().String()
	server.Config.Handler = httpx.NewHandler(httpx.Dependencies{
		Auth:          authSvc,
		Studio:        studioSvc,
		Packs:         catalog,
		Jobs:          runner,
		Assets:        files,
		PublicBaseURL: baseURL,
	})
	server.StartTLS()

	runCtx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- runner.Run(runCtx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("job runner exited with %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("job runner did not stop")
		}
		server.Close()
		db.Close()
	})

	adminClient := server.Client()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	adminClient.Jar = jar
	return &factoryEnv{
		t: t, ctx: ctx, db: db, baseURL: baseURL,
		admin: adminClient,
		// Separate client without the session cookie: the SDK proves the key alone.
		key: &http.Client{Transport: adminClient.Transport},
	}
}

func (e *factoryEnv) login() {
	e.t.Helper()
	e.mustAdmin(http.MethodPost, "/api/v1/admin/session", map[string]string{"password": ownerPassword}, nil, http.StatusNoContent)
	var created struct {
		Raw string `json:"raw"`
	}
	e.mustAdmin(http.MethodPost, "/api/v1/admin/api-keys", map[string]string{"name": "acceptance-client"}, &created, http.StatusCreated)
	if !strings.HasPrefix(created.Raw, "rsk_") {
		e.t.Fatalf("created key is malformed")
	}
	e.apiKey = created.Raw
}

func (e *factoryEnv) call(client *http.Client, method, target string, headers map[string]string, body []byte) (int, http.Header, []byte) {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(e.ctx, method, target, reader)
	if err != nil {
		e.t.Fatalf("build %s %s: %v", method, target, err)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", method, target, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read %s %s: %v", method, target, err)
	}
	return resp.StatusCode, resp.Header, data
}

func (e *factoryEnv) adminCall(method, target string, payload any) (int, []byte) {
	e.t.Helper()
	var body []byte
	headers := map[string]string{"Origin": e.baseURL}
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("encode %s %s: %v", method, target, err)
		}
		body = encoded
		headers["Content-Type"] = "application/json"
	}
	status, _, data := e.call(e.admin, method, e.baseURL+target, headers, body)
	return status, data
}

func (e *factoryEnv) mustAdmin(method, target string, payload, out any, want int) {
	e.t.Helper()
	status, data := e.adminCall(method, target, payload)
	if status != want {
		e.t.Fatalf("%s %s: status %d, want %d: %s", method, target, status, want, data)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			e.t.Fatalf("%s %s: decode response: %v (%s)", method, target, err, data)
		}
	}
}

func (e *factoryEnv) keyCall(method, target string) (int, http.Header, []byte) {
	e.t.Helper()
	return e.call(e.key, method, target, map[string]string{"X-Api-Key": e.apiKey}, nil)
}

func (e *factoryEnv) expectStatus(label string, status, want int, body []byte) {
	e.t.Helper()
	if status != want {
		e.t.Fatalf("%s: status %d, want %d: %s", label, status, want, body)
	}
}

func (e *factoryEnv) decodeKeyJSON(target string, out any) {
	e.t.Helper()
	status, _, data := e.keyCall(http.MethodGet, e.baseURL+target)
	e.expectStatus("GET "+target, status, http.StatusOK, data)
	if err := json.Unmarshal(data, out); err != nil {
		e.t.Fatalf("GET %s: decode response: %v (%s)", target, err, data)
	}
}

func (e *factoryEnv) uploadSource(packID string, imageBytes []byte) {
	e.t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "sheet.png")
	if err != nil {
		e.t.Fatalf("multipart file: %v", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		e.t.Fatalf("multipart write: %v", err)
	}
	if err := writer.Close(); err != nil {
		e.t.Fatalf("multipart close: %v", err)
	}
	status, _, data := e.call(e.admin, http.MethodPost, e.baseURL+"/api/v1/admin/packs/"+packID+"/source",
		map[string]string{"Content-Type": writer.FormDataContentType(), "Origin": e.baseURL}, buf.Bytes())
	if status != http.StatusOK {
		e.t.Fatalf("source upload: status %d: %s", status, data)
	}
}

func (e *factoryEnv) process(packID string) jobView {
	e.t.Helper()
	var job jobView
	e.mustAdmin(http.MethodPost, "/api/v1/admin/packs/"+packID+"/process", nil, &job, http.StatusAccepted)
	deadline := time.Now().Add(jobDeadline)
	for time.Now().Before(deadline) {
		var current jobView
		e.mustAdmin(http.MethodGet, "/api/v1/admin/jobs/"+job.ID, nil, &current, http.StatusOK)
		switch current.Status {
		case "succeeded":
			return current
		case "failed", "interrupted":
			e.t.Fatalf("job %s %s: %s %s", current.ID, current.Status, current.ErrorCode, current.ErrorMessage)
		}
		time.Sleep(pollInterval)
	}
	e.t.Fatalf("job %s did not finish before %s", job.ID, jobDeadline)
	return jobView{}
}

func (e *factoryEnv) captureVersion(packID, tag string) versionCapture {
	e.t.Helper()
	var list sdkList
	e.decodeKeyJSON("/api/v1/packs?tag="+tag, &list)

	var detail sdkDetail
	e.decodeKeyJSON("/api/v1/packs/"+packID, &detail)

	var meta sdkBundleMeta
	e.decodeKeyJSON("/api/v1/packs/"+packID+"/bundle", &meta)
	if meta.Version != detail.Version || meta.ChecksumSHA256 != detail.ChecksumSHA256 {
		e.t.Fatalf("bundle meta and detail disagree: %+v / %+v", meta, detail.sdkPack)
	}

	found := false
	for _, item := range list.Items {
		if item.ID == packID {
			found = item.Version == meta.Version && item.ChecksumSHA256 == meta.ChecksumSHA256
			break
		}
	}
	if !found {
		e.t.Fatalf("SDK list does not expose pack %s at version %d", packID, meta.Version)
	}

	status, headers, zipBytes := e.keyCall(http.MethodGet, meta.URL)
	e.expectStatus("bundle "+meta.URL, status, http.StatusOK, zipBytes)
	if !strings.HasPrefix(headers.Get("Content-Type"), "application/zip") {
		e.t.Fatalf("bundle content type = %q", headers.Get("Content-Type"))
	}
	members := verifyBundle(e.t, zipBytes, meta)

	capture := versionCapture{checksum: meta.ChecksumSHA256, bundleURL: meta.URL, zipBytes: zipBytes}
	if len(detail.Stickers) != 4 {
		e.t.Fatalf("version %d sticker count = %d, want 4", meta.Version, len(detail.Stickers))
	}
	for _, sticker := range detail.Stickers {
		capture.objectURLs = append(capture.objectURLs, sticker.FileURL)
		capture.thumbURLs = append(capture.thumbURLs, sticker.ThumbURL)

		status, headers, object := e.keyCall(http.MethodGet, sticker.FileURL)
		e.expectStatus("object "+sticker.FileURL, status, http.StatusOK, object)
		if !strings.HasPrefix(headers.Get("Content-Type"), "image/webp") {
			e.t.Fatalf("object content type = %q", headers.Get("Content-Type"))
		}
		member, ok := members[path.Base(sticker.FileURL)]
		if !ok || !bytes.Equal(object, member) {
			e.t.Fatalf("object %s does not match its bundle member", sticker.FileURL)
		}

		thumbFile := "thumbs/" + path.Base(sticker.ThumbURL)
		status, headers, thumb := e.keyCall(http.MethodGet, sticker.ThumbURL)
		e.expectStatus("thumb "+sticker.ThumbURL, status, http.StatusOK, thumb)
		if !strings.HasPrefix(headers.Get("Content-Type"), "image/webp") {
			e.t.Fatalf("thumb content type = %q", headers.Get("Content-Type"))
		}
		if member, ok := members[thumbFile]; !ok || !bytes.Equal(thumb, member) {
			e.t.Fatalf("thumb %s does not match its bundle member", sticker.ThumbURL)
		}
	}
	return capture
}

func (e *factoryEnv) tombstones(since string) tombstonePage {
	e.t.Helper()
	var page tombstonePage
	e.decodeKeyJSON("/api/v1/tombstones?since="+since, &page)
	return page
}

func (e *factoryEnv) expectGone(target string) {
	e.t.Helper()
	status, _, data := e.keyCall(http.MethodGet, target)
	e.expectStatus("revoked "+target, status, http.StatusGone, data)
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &failure); err != nil || failure.Error.Code != "pack_revoked" {
		e.t.Fatalf("revoked %s: error body %s", target, data)
	}
}

type versionCapture struct {
	checksum   string
	bundleURL  string
	zipBytes   []byte
	objectURLs []string
	thumbURLs  []string
}

func (v versionCapture) allURLs() []string {
	urls := append([]string{}, v.objectURLs...)
	urls = append(urls, v.thumbURLs...)
	return append(urls, v.bundleURL)
}

type packView struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type jobView struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type sdkPack struct {
	ID             string   `json:"id"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Author         string   `json:"author"`
	Tags           []string `json:"tags"`
	StickerCount   int      `json:"sticker_count"`
	Version        int      `json:"version"`
	ThumbnailURL   string   `json:"thumbnail_url"`
	ChecksumSHA256 string   `json:"checksum_sha256"`
}

type sdkSticker struct {
	File     string `json:"file"`
	FileURL  string `json:"file_url"`
	ThumbURL string `json:"thumb_url"`
}

type sdkDetail struct {
	sdkPack
	Stickers []sdkSticker `json:"stickers"`
}

type sdkList struct {
	Items []sdkPack `json:"items"`
}

type sdkBundleMeta struct {
	URL            string `json:"url"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	Version        int    `json:"version"`
}

type tombstonePage struct {
	Items []struct {
		PackID         string `json:"pack_id"`
		RevokedAt      string `json:"revoked_at"`
		ReasonCode     string `json:"reason_code"`
		ChecksumSHA256 string `json:"checksum_sha256"`
	} `json:"items"`
}

// verifyBundle checks the SDK checksum contract: SHA256 over the sticker bytes
// in pack.json manifest order (never the ZIP file hash) and full membership.
func verifyBundle(t *testing.T, zipBytes []byte, meta sdkBundleMeta) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("open bundle ZIP: %v", err)
	}
	members := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open bundle member %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read bundle member %s: %v", file.Name, err)
		}
		members[file.Name] = data
	}

	var manifest struct {
		Schema         string `json:"schema"`
		Version        int    `json:"version"`
		ChecksumSHA256 string `json:"checksum_sha256"`
		Stickers       []struct {
			File  string `json:"file"`
			Thumb string `json:"thumb"`
		} `json:"stickers"`
	}
	if err := json.Unmarshal(members["pack.json"], &manifest); err != nil {
		t.Fatalf("decode bundle pack.json: %v", err)
	}
	if manifest.Schema != assets.PackSchema {
		t.Fatalf("bundle schema = %q", manifest.Schema)
	}
	if manifest.Version != meta.Version || manifest.ChecksumSHA256 != meta.ChecksumSHA256 {
		t.Fatalf("bundle manifest identity = %d/%s, want %d/%s",
			manifest.Version, manifest.ChecksumSHA256, meta.Version, meta.ChecksumSHA256)
	}
	if len(manifest.Stickers) != 4 {
		t.Fatalf("bundle manifest stickers = %d, want 4", len(manifest.Stickers))
	}

	digest := sha256.New()
	for _, sticker := range manifest.Stickers {
		bytesOf, ok := members[sticker.File]
		if !ok {
			t.Fatalf("bundle is missing %s", sticker.File)
		}
		digest.Write(bytesOf)
		if _, ok := members[sticker.Thumb]; !ok {
			t.Fatalf("bundle is missing %s", sticker.Thumb)
		}
	}
	got := hex.EncodeToString(digest.Sum(nil))
	if got != meta.ChecksumSHA256 {
		t.Fatalf("manifest-order checksum = %s, bundle meta = %s", got, meta.ChecksumSHA256)
	}
	zipSum := sha256.Sum256(zipBytes)
	if hex.EncodeToString(zipSum[:]) == meta.ChecksumSHA256 {
		t.Fatalf("checksum equals the ZIP hash; it must hash sticker bytes")
	}
	if _, ok := members["thumb.webp"]; !ok {
		t.Fatalf("bundle is missing thumb.webp")
	}
	return members
}

var (
	quadrantColorsA = [4]color.RGBA{
		{R: 220, G: 40, B: 40, A: 255},
		{R: 40, G: 180, B: 60, A: 255},
		{R: 40, G: 80, B: 220, A: 255},
		{R: 230, G: 200, B: 40, A: 255},
	}
	quadrantColorsB = [4]color.RGBA{
		{R: 30, G: 120, B: 200, A: 255},
		{R: 200, G: 60, B: 30, A: 255},
		{R: 20, G: 160, B: 120, A: 255},
		{R: 240, G: 140, B: 20, A: 255},
	}
)

// sheetPNG renders an owned 1024x1024 white sheet with one circle per 2x2 cell.
// The fixture is deterministic and is not evidence of AI image quality.
func sheetPNG(colors [4]color.RGBA) []byte {
	const (
		size   = 1024
		radius = 150
	)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	centers := [4]image.Point{{X: 256, Y: 256}, {X: 768, Y: 256}, {X: 256, Y: 768}, {X: 768, Y: 768}}
	for index, fill := range colors {
		cx, cy := centers[index].X, centers[index].Y
		for y := cy - radius; y <= cy+radius; y++ {
			for x := cx - radius; x <= cx+radius; x++ {
				dx, dy := x-cx, y-cy
				if dx*dx+dy*dy <= radius*radius {
					img.SetRGBA(x, y, fill)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
