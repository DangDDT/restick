package httpx

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/eykh/restick/api_svc/internal/studio"
)

const (
	adminBaseURL    = "https://sticker.example"
	adminOriginKey  = "Origin"
	jsonContentType = "application/json"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

type adminFixture struct {
	t       *testing.T
	root    string
	db      *sql.DB
	auth    *auth.Service
	studio  *studio.Service
	packs   *packs.Service
	files   *assets.Store
	jobs    *jobs.Runner
	handler http.Handler
	clock   *testClock
	webDir  string
	session string
}

func newAdminFixture(t *testing.T) *adminFixture {
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
	studioSvc := studio.NewService(db, files)
	catalog := packs.NewService(db)
	runner := jobs.NewRunner(db, files, nil, time.Minute)
	webDir := filepath.Join(root, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	trusted, err := auth.ParseTrustedCIDRs([]string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("ParseTrustedCIDRs: %v", err)
	}
	clock := newTestClock()
	handler := newHandler(Dependencies{
		Auth:              authSvc,
		Studio:            studioSvc,
		Packs:             catalog,
		Jobs:              runner,
		Assets:            files,
		PublicBaseURL:     adminBaseURL,
		TrustedProxyCIDRs: trusted,
		WebDir:            webDir,
		OpenAPIPath:       filepath.Join(root, "v1.yaml"),
	}, clock.Now)
	return &adminFixture{
		t: t, root: root, db: db, auth: authSvc, studio: studioSvc, packs: catalog,
		files: files, jobs: runner, handler: handler, clock: clock, webDir: webDir,
	}
}

func (f *adminFixture) request(method, target string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.RemoteAddr = "192.0.2.10:4000"
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func (f *adminFixture) login() {
	f.t.Helper()
	rec := f.request(http.MethodPost, "/api/v1/admin/session", []byte(`{"password":"owner-password"}`),
		map[string]string{"Content-Type": jsonContentType, adminOriginKey: adminBaseURL})
	if rec.Code != http.StatusNoContent {
		f.t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		f.t.Fatalf("login cookies = %d", len(cookies))
	}
	f.session = cookies[0].Value
}

func (f *adminFixture) adminHeaders() map[string]string {
	return map[string]string{
		"Cookie":       adminCookieName + "=" + f.session,
		adminOriginKey: adminBaseURL,
		"Content-Type": jsonContentType,
	}
}

func (f *adminFixture) mutate(method, target string, payload any) *httptest.ResponseRecorder {
	f.t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			f.t.Fatal(err)
		}
	}
	return f.request(method, target, body, f.adminHeaders())
}

func (f *adminFixture) get(target string) *httptest.ResponseRecorder {
	f.t.Helper()
	return f.request(http.MethodGet, target, nil, map[string]string{
		"Cookie": adminCookieName + "=" + f.session,
	})
}

func (f *adminFixture) upload(target, field, filename string, data []byte) *httptest.ResponseRecorder {
	f.t.Helper()
	body, contentType := multipartBody(f.t, field, filename, data)
	headers := f.adminHeaders()
	headers["Content-Type"] = contentType
	return f.request(http.MethodPost, target, body, headers)
}

func multipartBody(t *testing.T, field, filename string, data []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if field != "" {
		part, err := writer.CreateFormFile(field, filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes(), writer.FormDataContentType()
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 40, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return payload
}

func decodeInto(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	payload := decodeMap(t, rec)
	envelope, _ := payload["error"].(map[string]any)
	code, _ := envelope["code"].(string)
	return code
}

func (f *adminFixture) tableCounts() map[string]int {
	f.t.Helper()
	tables := []string{
		"fragment_groups", "fragments", "characters", "designs", "packs",
		"api_keys", "admin_sessions", "reference_assets", "pack_tombstones", "processing_jobs",
	}
	counts := map[string]int{}
	for _, table := range tables {
		var count int
		if err := f.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			f.t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = count
	}
	return counts
}

// readyPack writes a committed v1 row so revoke has a real version to end.
func (f *adminFixture) readyPack(slug string) packs.Pack {
	f.t.Helper()
	ctx := context.Background()
	pack, err := f.packs.Create(ctx, packs.PackDraft{Slug: slug, Name: "Pack " + slug, Author: "Tester", Tags: []string{"demo"}})
	if err != nil {
		f.t.Fatalf("packs.Create: %v", err)
	}
	checksum := strings.Repeat("b", 64)
	outputDir := fmt.Sprintf("packs/%s/v1", pack.ID)
	if _, err := f.db.Exec(`INSERT INTO pack_versions
		(pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, 1, ?, ?, 1, '{}', ?, 10, ?, ?)`,
		pack.ID, "job-"+pack.ID, checksum, fmt.Sprintf("bundles/%s/v1.zip", pack.ID),
		outputDir+"/thumb.webp", outputDir); err != nil {
		f.t.Fatalf("insert version: %v", err)
	}
	if _, err := f.db.Exec(`UPDATE packs SET status = 'ready', current_version = 1 WHERE id = ?`, pack.ID); err != nil {
		f.t.Fatalf("mark ready: %v", err)
	}
	ready, err := f.packs.Get(ctx, pack.ID)
	if err != nil {
		f.t.Fatalf("packs.Get: %v", err)
	}
	return ready
}

// TestClientKeyCannotMutateAnyAdminRoute proves a real client key cannot reach
// any admin read or write route, and that rejected writes change no rows.
func TestClientKeyCannotMutateAnyAdminRoute(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	key, err := f.auth.CreateKey(ctx, "sdk", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	id := "00000000-0000-0000-0000-000000000000"
	imageBody, imageType := multipartBody(t, "file", "sheet.png", testPNG(t, 2, 2))
	jsonBody := func(payload string) []byte { return []byte(payload) }

	routes := []struct {
		method      string
		target      string
		body        []byte
		contentType string
	}{
		{http.MethodGet, "/api/v1/admin/session", nil, ""},
		{http.MethodGet, "/api/v1/admin/fragment-groups", nil, ""},
		{http.MethodPost, "/api/v1/admin/fragment-groups", jsonBody(`{"name":"Species"}`), jsonContentType},
		{http.MethodPut, "/api/v1/admin/fragment-groups/" + id, jsonBody(`{"name":"Species"}`), jsonContentType},
		{http.MethodDelete, "/api/v1/admin/fragment-groups/" + id, nil, ""},
		{http.MethodGet, "/api/v1/admin/fragments", nil, ""},
		{http.MethodPost, "/api/v1/admin/fragments", jsonBody(`{"group_id":"g","name":"Tail","prompt":"a tail","scope":"identity"}`), jsonContentType},
		{http.MethodGet, "/api/v1/admin/fragments/" + id, nil, ""},
		{http.MethodPut, "/api/v1/admin/fragments/" + id, jsonBody(`{"group_id":"g","name":"Tail","prompt":"a tail","scope":"identity"}`), jsonContentType},
		{http.MethodDelete, "/api/v1/admin/fragments/" + id, nil, ""},
		{http.MethodGet, "/api/v1/admin/characters", nil, ""},
		{http.MethodPost, "/api/v1/admin/characters", jsonBody(`{"name":"Mèo"}`), jsonContentType},
		{http.MethodGet, "/api/v1/admin/characters/" + id, nil, ""},
		{http.MethodPut, "/api/v1/admin/characters/" + id, jsonBody(`{"name":"Mèo"}`), jsonContentType},
		{http.MethodDelete, "/api/v1/admin/characters/" + id, nil, ""},
		{http.MethodPost, "/api/v1/admin/reference-assets", imageBody, imageType},
		{http.MethodGet, "/api/v1/admin/reference-assets/" + id, nil, ""},
		{http.MethodGet, "/api/v1/admin/designs", nil, ""},
		{http.MethodPost, "/api/v1/admin/designs", jsonBody(`{"name":"A","rows":1,"columns":1,"cells":[{"action":"wave"}]}`), jsonContentType},
		{http.MethodPost, "/api/v1/admin/designs/compile", jsonBody(`{"rows":1,"columns":1,"cells":[{"action":"wave"}]}`), jsonContentType},
		{http.MethodPost, "/api/v1/admin/designs/suggest", jsonBody(`{"rows":1,"columns":1,"cells":[{"action":"wave"}]}`), jsonContentType},
		{http.MethodGet, "/api/v1/admin/designs/" + id, nil, ""},
		{http.MethodPut, "/api/v1/admin/designs/" + id, jsonBody(`{"name":"A","rows":1,"columns":1,"cells":[{"action":"wave"}]}`), jsonContentType},
		{http.MethodDelete, "/api/v1/admin/designs/" + id, nil, ""},
		{http.MethodGet, "/api/v1/admin/packs", nil, ""},
		{http.MethodPost, "/api/v1/admin/packs", jsonBody(`{"slug":"stolen","name":"Stolen"}`), jsonContentType},
		{http.MethodGet, "/api/v1/admin/packs/" + id, nil, ""},
		{http.MethodPut, "/api/v1/admin/packs/" + id, jsonBody(`{"name":"Stolen","author":"x","tags":[]}`), jsonContentType},
		{http.MethodDelete, "/api/v1/admin/packs/" + id, nil, ""},
		{http.MethodPost, "/api/v1/admin/packs/" + id + "/source", imageBody, imageType},
		{http.MethodPost, "/api/v1/admin/packs/" + id + "/process", nil, ""},
		{http.MethodPost, "/api/v1/admin/packs/" + id + "/revoke", nil, ""},
		{http.MethodGet, "/api/v1/admin/jobs/" + id, nil, ""},
		{http.MethodGet, "/api/v1/admin/api-keys", nil, ""},
		{http.MethodPost, "/api/v1/admin/api-keys", jsonBody(`{"name":"stolen"}`), jsonContentType},
		{http.MethodPost, "/api/v1/admin/api-keys/" + id + "/rotate", nil, ""},
		{http.MethodDelete, "/api/v1/admin/api-keys/" + id, nil, ""},
	}

	before := f.tableCounts()
	for _, route := range routes {
		headers := map[string]string{"X-Api-Key": key.Raw, adminOriginKey: adminBaseURL}
		if route.contentType != "" {
			headers["Content-Type"] = route.contentType
		}
		rec := f.request(route.method, route.target, route.body, headers)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with client key = %d, want 401 (%s)", route.method, route.target, rec.Code, rec.Body.String())
			continue
		}
		if code := errorCode(t, rec); code != "unauthorized" {
			t.Errorf("%s %s code = %q, want unauthorized", route.method, route.target, code)
		}
	}
	after := f.tableCounts()
	for table, count := range before {
		if after[table] != count {
			t.Errorf("%s rows changed from %d to %d after rejected key requests", table, count, after[table])
		}
	}
	if after["api_keys"] != 1 {
		t.Errorf("api_keys rows = %d, want the single client key", after["api_keys"])
	}

	// The same key must still read the SDK catalog (read-only by design).
	rec := f.request(http.MethodGet, "/api/v1/packs", nil, map[string]string{"X-Api-Key": key.Raw})
	if rec.Code != http.StatusOK {
		t.Fatalf("client key SDK read = %d: %s", rec.Code, rec.Body.String())
	}
}

// TestMissingForeignOriginRejected covers the Origin rule for login and for
// unsafe admin requests.
func TestMissingForeignOriginRejected(t *testing.T) {
	f := newAdminFixture(t)
	login := func(origin string) *httptest.ResponseRecorder {
		headers := map[string]string{"Content-Type": jsonContentType}
		if origin != "" {
			headers[adminOriginKey] = origin
		}
		return f.request(http.MethodPost, "/api/v1/admin/session", []byte(`{"password":"owner-password"}`), headers)
	}

	if rec := login(""); rec.Code != http.StatusForbidden {
		t.Fatalf("login without origin = %d, want 403", rec.Code)
	} else if code := errorCode(t, rec); code != "origin_not_allowed" {
		t.Fatalf("login without origin code = %q", code)
	}
	if rec := login("https://evil.example"); rec.Code != http.StatusForbidden {
		t.Fatalf("login with foreign origin = %d, want 403", rec.Code)
	}
	rec := login(adminBaseURL)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login with origin = %d: %s", rec.Code, rec.Body.String())
	}
	f.session = rec.Result().Cookies()[0].Value

	cookie := map[string]string{"Cookie": adminCookieName + "=" + f.session}

	// Safe methods do not need an Origin.
	if rec := f.get("/api/v1/admin/fragment-groups"); rec.Code != http.StatusOK {
		t.Fatalf("GET groups = %d", rec.Code)
	}

	body := []byte(`{"name":"Species"}`)
	for _, origin := range []string{"", "https://evil.example"} {
		headers := map[string]string{"Content-Type": jsonContentType, "Cookie": cookie["Cookie"]}
		if origin != "" {
			headers[adminOriginKey] = origin
		}
		rec := f.request(http.MethodPost, "/api/v1/admin/fragment-groups", body, headers)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("POST group with origin %q = %d, want 403", origin, rec.Code)
		}
		if code := errorCode(t, rec); code != "origin_not_allowed" {
			t.Fatalf("POST group with origin %q code = %q", origin, code)
		}
	}
	var groups int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM fragment_groups`).Scan(&groups); err != nil {
		t.Fatal(err)
	}
	if groups != 0 {
		t.Fatalf("fragment_groups = %d after rejected writes", groups)
	}

	if rec := f.mutate(http.MethodPost, "/api/v1/admin/fragment-groups", map[string]any{"name": "Species"}); rec.Code != http.StatusCreated {
		t.Fatalf("POST group with origin = %d: %s", rec.Code, rec.Body.String())
	}

	// Logout is unsafe too.
	headers := map[string]string{"Cookie": cookie["Cookie"], "Content-Type": jsonContentType}
	if rec := f.request(http.MethodDelete, "/api/v1/admin/session", nil, headers); rec.Code != http.StatusForbidden {
		t.Fatalf("logout without origin = %d, want 403", rec.Code)
	}
}

// TestCookieFlagsAndLogout pins the session cookie attributes and the
// server-side revoke.
func TestCookieFlagsAndLogout(t *testing.T) {
	f := newAdminFixture(t)
	rec := f.request(http.MethodPost, "/api/v1/admin/session", []byte(`{"password":"owner-password"}`),
		map[string]string{"Content-Type": jsonContentType, adminOriginKey: adminBaseURL})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login = %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("login body = %q, want empty", rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login cookies = %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "__Host-restick_admin" {
		t.Errorf("cookie name = %q", cookie.Name)
	}
	if cookie.Value == "" || len(cookie.Value) != 64 {
		t.Errorf("cookie value = %q, want 64 hex chars", cookie.Value)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie flags httpOnly=%v secure=%v sameSite=%v", cookie.HttpOnly, cookie.Secure, cookie.SameSite)
	}
	if cookie.Path != "/" || cookie.Domain != "" {
		t.Errorf("cookie scope path=%q domain=%q", cookie.Path, cookie.Domain)
	}
	if cookie.MaxAge != int(auth.SessionTTL/time.Second) {
		t.Errorf("cookie maxAge = %d", cookie.MaxAge)
	}
	if strings.Contains(rec.Body.String(), cookie.Value) {
		t.Fatal("raw session token leaked into the login body")
	}

	f.session = cookie.Value
	session := f.get("/api/v1/admin/session")
	if session.Code != http.StatusOK {
		t.Fatalf("GET session = %d: %s", session.Code, session.Body.String())
	}
	payload := decodeMap(t, session)
	if payload["username"] != auth.OwnerUsername {
		t.Errorf("username = %v", payload["username"])
	}
	if strings.Contains(session.Body.String(), cookie.Value) {
		t.Fatal("raw session token leaked into the session body")
	}

	logout := f.mutate(http.MethodDelete, "/api/v1/admin/session", nil)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout = %d: %s", logout.Code, logout.Body.String())
	}
	cleared := logout.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != cookie.Name || cleared[0].Value != "" || cleared[0].MaxAge >= 0 {
		t.Fatalf("logout cookie = %+v", cleared)
	}
	if rec := f.get("/api/v1/admin/session"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET session after logout = %d, want 401", rec.Code)
	}
	if err := f.auth.ValidateSession(context.Background(), cookie.Value, f.clock.Now()); err == nil {
		t.Fatal("server still accepts the logged-out session token")
	}
}

// TestLoginRateLimitWindow drives the limiter with an injected clock so the
// 15 minute window needs no sleep.
func TestLoginRateLimitWindow(t *testing.T) {
	f := newAdminFixture(t)
	attempt := func(remoteAddr, forwardedFor, password string) *httptest.ResponseRecorder {
		headers := map[string]string{"Content-Type": jsonContentType, adminOriginKey: adminBaseURL}
		if forwardedFor != "" {
			headers["X-Forwarded-For"] = forwardedFor
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/session",
			strings.NewReader(fmt.Sprintf(`{"password":%q}`, password)))
		req.RemoteAddr = remoteAddr
		for name, value := range headers {
			req.Header.Set(name, value)
		}
		rec := httptest.NewRecorder()
		f.handler.ServeHTTP(rec, req)
		return rec
	}

	for i := range loginMaxFailures {
		if rec := attempt("10.0.0.1:1111", "", "wrong-password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d = %d, want 401", i+1, rec.Code)
		}
	}
	if rec := attempt("10.0.0.1:1111", "", "owner-password"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("blocked attempt = %d, want 429", rec.Code)
	} else if code := errorCode(t, rec); code != "rate_limited" {
		t.Fatalf("blocked attempt code = %q", code)
	}
	// Forwarded headers from an untrusted peer must not open a fresh bucket.
	if rec := attempt("10.0.0.1:1111", "203.0.113.9", "owner-password"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("forged X-Forwarded-For attempt = %d, want 429", rec.Code)
	}
	// Another source has its own budget.
	if rec := attempt("10.0.0.2:2222", "", "owner-password"); rec.Code != http.StatusNoContent {
		t.Fatalf("second source = %d, want 204", rec.Code)
	}

	f.clock.Advance(14 * time.Minute)
	if rec := attempt("10.0.0.1:1111", "", "owner-password"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt inside the window = %d, want 429", rec.Code)
	}
	f.clock.Advance(time.Minute)
	if rec := attempt("10.0.0.1:1111", "", "owner-password"); rec.Code != http.StatusNoContent {
		t.Fatalf("attempt after the window = %d, want 204", rec.Code)
	}
}

// TestJSONAndMultipartLimits proves every documented transport bound.
func TestJSONAndMultipartLimits(t *testing.T) {
	f := newAdminFixture(t)
	f.login()

	oversized := fmt.Sprintf(`{"name":%q}`, strings.Repeat("a", maxJSONBody))
	rec := f.request(http.MethodPost, "/api/v1/admin/fragment-groups", []byte(oversized), f.adminHeaders())
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized JSON = %d, want 413", rec.Code)
	}
	if code := errorCode(t, rec); code != "payload_too_large" {
		t.Fatalf("oversized JSON code = %q", code)
	}

	headers := f.adminHeaders()
	headers["Content-Type"] = "text/plain"
	if rec := f.request(http.MethodPost, "/api/v1/admin/fragment-groups", []byte(`{"name":"x"}`), headers); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("wrong content type = %d, want 422", rec.Code)
	}
	if rec := f.mutate(http.MethodPost, "/api/v1/admin/fragment-groups", map[string]any{"name": "x", "extra": 1}); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field = %d, want 400", rec.Code)
	}
	if rec := f.request(http.MethodPost, "/api/v1/admin/fragment-groups", []byte(`{"name":"x"}{"name":"y"}`), f.adminHeaders()); rec.Code != http.StatusBadRequest {
		t.Fatalf("two documents = %d, want 400", rec.Code)
	}

	// Total multipart body over 8 MiB + 64 KiB, spread over skipped parts.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for range 9 {
		if err := writer.WriteField("junk", strings.Repeat("a", 1<<20)); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", "sheet.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(testPNG(t, 2, 2)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	wide := f.adminHeaders()
	wide["Content-Type"] = writer.FormDataContentType()
	if rec := f.request(http.MethodPost, "/api/v1/admin/reference-assets", buf.Bytes(), wide); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized multipart = %d, want 413", rec.Code)
	}

	if rec := f.upload("/api/v1/admin/reference-assets", "file", "sheet.png", make([]byte, 8<<20+1)); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized file part = %d, want 413", rec.Code)
	}

	headers = f.adminHeaders()
	headers["Content-Type"] = jsonContentType
	if rec := f.request(http.MethodPost, "/api/v1/admin/reference-assets", []byte(`{}`), headers); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("multipart route with json = %d, want 422", rec.Code)
	}
	if rec := f.upload("/api/v1/admin/reference-assets", "", "sheet.png", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("multipart without file part = %d, want 400", rec.Code)
	}

	var groups int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM fragment_groups`).Scan(&groups); err != nil {
		t.Fatal(err)
	}
	if groups != 0 {
		t.Fatalf("fragment_groups = %d after rejected bodies", groups)
	}

	// The same route still accepts a valid upload.
	rec = f.upload("/api/v1/admin/reference-assets", "file", "ok.png", testPNG(t, 3, 3))
	if rec.Code != http.StatusCreated {
		t.Fatalf("valid upload = %d: %s", rec.Code, rec.Body.String())
	}
}

// TestPrivateReferenceNotPublic proves reference bytes have exactly one reader:
// the owner-session route.
func TestPrivateReferenceNotPublic(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	f.login()
	pngBytes := testPNG(t, 4, 5)
	rec := f.upload("/api/v1/admin/reference-assets", "file", "ref.png", pngBytes)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d: %s", rec.Code, rec.Body.String())
	}
	var asset studio.ReferenceAsset
	decodeInto(t, rec, &asset)
	if asset.ID == "" || asset.ContentType != "image/png" {
		t.Fatalf("asset = %+v", asset)
	}

	preview := f.get("/api/v1/admin/reference-assets/" + asset.ID)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview = %d: %s", preview.Code, preview.Body.String())
	}
	if preview.Header().Get("Content-Type") != "image/png" {
		t.Errorf("preview content type = %q", preview.Header().Get("Content-Type"))
	}
	if preview.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("preview cache = %q", preview.Header().Get("Cache-Control"))
	}
	if !bytes.Equal(preview.Body.Bytes(), pngBytes) {
		t.Fatal("preview bytes differ from the uploaded image")
	}

	key, err := f.auth.CreateKey(ctx, "sdk", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	objectPath := "/v1/objects/sources/" + asset.ID + "/original.png"
	if rec := f.request(http.MethodGet, objectPath, nil, map[string]string{"X-Api-Key": key.Raw}); rec.Code != http.StatusNotFound {
		t.Fatalf("public object for reference = %d, want 404", rec.Code)
	}
	if rec := f.request(http.MethodGet, objectPath, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous reference object = %d, want 401", rec.Code)
	}
	if rec := f.request(http.MethodGet, "/api/v1/admin/reference-assets/"+asset.ID, nil, map[string]string{"X-Api-Key": key.Raw}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("client key preview = %d, want 401", rec.Code)
	}
	catalog := f.request(http.MethodGet, "/api/v1/packs", nil, map[string]string{"X-Api-Key": key.Raw})
	if catalog.Code != http.StatusOK {
		t.Fatalf("catalog = %d", catalog.Code)
	}
	if strings.Contains(catalog.Body.String(), asset.ID) {
		t.Fatal("catalog leaked a private reference id")
	}
}

// TestRemovedCommunityRoutesNotRegistered keeps the retired community API off
// the new handler: JSON 404, never the HTML shell.
func TestRemovedCommunityRoutesNotRegistered(t *testing.T) {
	f := newAdminFixture(t)
	routes := []struct{ method, target string }{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodGet, "/api/v1/me"},
		{http.MethodGet, "/api/v1/me/packs"},
		{http.MethodGet, "/api/v1/me/api-keys"},
		{http.MethodPost, "/api/v1/me/api-keys"},
		{http.MethodPost, "/api/v1/packs/00000000-0000-0000-0000-000000000000/submit"},
		{http.MethodPost, "/api/v1/packs/00000000-0000-0000-0000-000000000000/confirm"},
		{http.MethodPost, "/api/v1/packs/00000000-0000-0000-0000-000000000000/revoke"},
		{http.MethodGet, "/api/v1/admin/published"},
		{http.MethodGet, "/api/v1/admin/analytics/rejections"},
		{http.MethodGet, "/api/v1/admin/packs/00000000-0000-0000-0000-000000000000/queue"},
	}
	for _, route := range routes {
		rec := f.request(route.method, route.target, nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", route.method, route.target, rec.Code)
			continue
		}
		if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
			t.Errorf("%s %s content type = %q, want JSON", route.method, route.target, contentType)
			continue
		}
		if code := errorCode(t, rec); code != "not_found" {
			t.Errorf("%s %s code = %q", route.method, route.target, code)
		}
	}

	// A method that no pattern accepts still answers the JSON 404 envelope.
	rec := f.request(http.MethodPatch, "/api/v1/admin/session", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PATCH session = %d, want 404", rec.Code)
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("PATCH session content type = %q", rec.Header().Get("Content-Type"))
	}
}

// TestStaticExportedUIRouting proves the exported UI is served and that missing
// file/API paths never fall back to HTML.
func TestStaticExportedUIRouting(t *testing.T) {
	f := newAdminFixture(t)
	write := func(rel, body string) {
		full := filepath.Join(f.webDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("factory.html", "<html>factory</html>")
	write("login/index.html", "<html>login</html>")
	write("_next/static/app.js", "console.log('ui')")

	root := f.request(http.MethodGet, "/", nil, nil)
	if root.Code != http.StatusTemporaryRedirect || root.Header().Get("Location") != "/factory" {
		t.Fatalf("root = %d location=%q", root.Code, root.Header().Get("Location"))
	}
	if rec := f.request(http.MethodGet, "/factory", nil, nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "factory") {
		t.Fatalf("/factory = %d %q", rec.Code, rec.Body.String())
	}
	if rec := f.request(http.MethodGet, "/login/", nil, nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "login") {
		t.Fatalf("/login/ = %d %q", rec.Code, rec.Body.String())
	}
	if rec := f.request(http.MethodGet, "/_next/static/app.js", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("/_next/static/app.js = %d", rec.Code)
	}
	missing := f.request(http.MethodGet, "/missing.js", nil, nil)
	if missing.Code != http.StatusNotFound || strings.Contains(missing.Body.String(), "factory") {
		t.Fatalf("missing file = %d %q", missing.Code, missing.Body.String())
	}
	for _, target := range []string{"/api/v1/missing", "/v1/missing"} {
		rec := f.request(http.MethodGet, target, nil, nil)
		if rec.Code != http.StatusNotFound || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("%s = %d %q", target, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
}

// TestAdminFactoryLifecycle runs the real admin workflow against SQLite, the
// filesystem store and the domain services.
func TestAdminFactoryLifecycle(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	f.login()

	// Library: group, fragment, update, archive.
	rec := f.mutate(http.MethodPost, "/api/v1/admin/fragment-groups", map[string]any{"name": "Species", "sort_order": 1})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group = %d: %s", rec.Code, rec.Body.String())
	}
	var group studio.Group
	decodeInto(t, rec, &group)

	rec = f.mutate(http.MethodPost, "/api/v1/admin/fragments", map[string]any{
		"group_id": group.ID, "name": "Mèo", "name_en": "Cat", "prompt": "a small cat", "scope": "identity",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create fragment = %d: %s", rec.Code, rec.Body.String())
	}
	var fragment studio.Fragment
	decodeInto(t, rec, &fragment)

	rec = f.mutate(http.MethodPut, "/api/v1/admin/fragments/"+fragment.ID, map[string]any{
		"group_id": group.ID, "name": "Mèo con", "name_en": "Kitten", "prompt": "a small cat", "scope": "identity",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update fragment = %d: %s", rec.Code, rec.Body.String())
	}
	var updated studio.Fragment
	decodeInto(t, rec, &updated)
	if updated.Name != "Mèo con" {
		t.Fatalf("updated name = %q", updated.Name)
	}

	rec = f.get("/api/v1/admin/fragments?q=Kitten&scope=identity&page=1&page_size=10")
	if rec.Code != http.StatusOK {
		t.Fatalf("list fragments = %d", rec.Code)
	}
	var fragments studio.FragmentPage
	decodeInto(t, rec, &fragments)
	if fragments.Total != 1 || len(fragments.Items) != 1 || fragments.Items[0].ID != fragment.ID {
		t.Fatalf("fragment page = %+v", fragments)
	}

	// Characters: private reference upload then a character pointing at it.
	rec = f.upload("/api/v1/admin/reference-assets", "file", "ref.png", testPNG(t, 6, 7))
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload reference = %d: %s", rec.Code, rec.Body.String())
	}
	var reference studio.ReferenceAsset
	decodeInto(t, rec, &reference)

	rec = f.mutate(http.MethodPost, "/api/v1/admin/characters", map[string]any{
		"name": "Mèo Mun", "description": "black cat", "identity_notes": "white whiskers",
		"reference_asset_id": reference.ID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create character = %d: %s", rec.Code, rec.Body.String())
	}
	var character studio.Character
	decodeInto(t, rec, &character)
	if character.ReferenceAssetID != reference.ID {
		t.Fatalf("character reference = %q", character.ReferenceAssetID)
	}
	if rec := f.get("/api/v1/admin/characters/" + character.ID); rec.Code != http.StatusOK {
		t.Fatalf("get character = %d", rec.Code)
	}

	// Designs: omitted outline_width binds to the 4 px default, an explicit 0
	// keeps meaning "no outline", and the compiled prompt proves it downstream.
	rec = f.mutate(http.MethodPost, "/api/v1/admin/designs/compile", map[string]any{
		"rows": 1, "columns": 1, "cells": []map[string]any{{"action": "wave"}},
		"fragment_ids": []string{fragment.ID}, "character_id": character.ID, "outline_color": "#FFFFFF",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("compile default = %d: %s", rec.Code, rec.Body.String())
	}
	var defaultCompile compiledDesign
	decodeInto(t, rec, &defaultCompile)
	if defaultCompile.Snapshot.OutlineWidth != studio.DefaultOutlineWidth {
		t.Fatalf("default outline width = %d, want %d", defaultCompile.Snapshot.OutlineWidth, studio.DefaultOutlineWidth)
	}
	if !strings.Contains(defaultCompile.CompiledPrompt, "4 px #FFFFFF outline") {
		t.Fatalf("default prompt lacks the 4 px outline: %q", defaultCompile.CompiledPrompt)
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/designs/compile", map[string]any{
		"rows": 1, "columns": 1, "cells": []map[string]any{{"action": "wave"}}, "outline_width": 0,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("compile zero = %d: %s", rec.Code, rec.Body.String())
	}
	var zeroCompile compiledDesign
	decodeInto(t, rec, &zeroCompile)
	if zeroCompile.Snapshot.OutlineWidth != 0 {
		t.Fatalf("explicit zero outline width = %d", zeroCompile.Snapshot.OutlineWidth)
	}
	if !strings.Contains(zeroCompile.CompiledPrompt, "Outline: none is added") {
		t.Fatalf("zero prompt lacks the no-outline line: %q", zeroCompile.CompiledPrompt)
	}
	if zeroCompile.CompiledPrompt == defaultCompile.CompiledPrompt {
		t.Fatal("omitted and explicit-zero drafts compiled to the same prompt")
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/designs", map[string]any{
		"name": "Design A", "rows": 1, "columns": 1, "cells": []map[string]any{{"action": "wave"}},
		"character_id": character.ID, "fragment_ids": []string{fragment.ID},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("save design = %d: %s", rec.Code, rec.Body.String())
	}
	var design studio.Design
	decodeInto(t, rec, &design)
	if design.OutlineWidth != studio.DefaultOutlineWidth || design.Snapshot.OutlineWidth != studio.DefaultOutlineWidth {
		t.Fatalf("saved design outline = %d/%d, want %d", design.OutlineWidth, design.Snapshot.OutlineWidth, studio.DefaultOutlineWidth)
	}
	if design.Prompt == "" {
		t.Fatal("saved design has no compiled prompt")
	}

	rec = f.mutate(http.MethodPut, "/api/v1/admin/designs/"+design.ID, map[string]any{
		"id": "00000000-0000-0000-0000-000000000000", "name": "Design A", "rows": 1, "columns": 1,
		"cells": []map[string]any{{"action": "wave"}}, "outline_width": 0,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update design = %d: %s", rec.Code, rec.Body.String())
	}
	var updatedDesign studio.Design
	decodeInto(t, rec, &updatedDesign)
	if updatedDesign.ID != design.ID || updatedDesign.OutlineWidth != 0 {
		t.Fatalf("updated design id/outline = %q/%d", updatedDesign.ID, updatedDesign.OutlineWidth)
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/designs/suggest", map[string]any{
		"name": "Suggested", "rows": 1, "columns": 1, "cells": []map[string]any{{"action": "wave"}},
		"group_ids": []string{group.ID},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("suggest = %d: %s", rec.Code, rec.Body.String())
	}
	var suggested studio.Draft
	decodeInto(t, rec, &suggested)
	if len(suggested.FragmentIDs) != 1 || suggested.FragmentIDs[0] != fragment.ID {
		t.Fatalf("suggested fragments = %v", suggested.FragmentIDs)
	}

	// Packs: create draft, list with status filter, upload source, enqueue job.
	rec = f.mutate(http.MethodPost, "/api/v1/admin/packs", map[string]any{
		"slug": "meo-mun", "name": "Mèo Mun", "author": "Operator", "tags": []string{"cat"}, "design_id": design.ID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pack = %d: %s", rec.Code, rec.Body.String())
	}
	var pack adminPack
	decodeInto(t, rec, &pack)
	if pack.Status != packs.StatusDraft || pack.Slug != "meo-mun" {
		t.Fatalf("created pack = %+v", pack)
	}
	if pack.Source != nil {
		t.Fatalf("new pack source = %+v, want null", pack.Source)
	}

	rec = f.get("/api/v1/admin/packs?status=draft&q=meo&page=1&page_size=10")
	if rec.Code != http.StatusOK {
		t.Fatalf("list packs = %d", rec.Code)
	}
	var packPage struct {
		Items    []adminPack `json:"items"`
		Page     int         `json:"page"`
		PageSize int         `json:"page_size"`
		Total    int         `json:"total"`
	}
	decodeInto(t, rec, &packPage)
	if packPage.Total != 1 || len(packPage.Items) != 1 || packPage.Items[0].ID != pack.ID {
		t.Fatalf("draft page = %+v", packPage)
	}

	rec = f.upload("/api/v1/admin/packs/"+pack.ID+"/source", "file", "sheet.png", testPNG(t, 64, 64))
	if rec.Code != http.StatusOK {
		t.Fatalf("upload source = %d: %s", rec.Code, rec.Body.String())
	}
	var sourced adminPack
	decodeInto(t, rec, &sourced)
	if sourced.Source == nil || sourced.Source.Width != 64 {
		t.Fatalf("pack source = %+v", sourced.Source)
	}

	rec = f.get("/api/v1/admin/packs/" + pack.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("pack detail = %d: %s", rec.Code, rec.Body.String())
	}
	detail := decodeMap(t, rec)
	if detail["status"] != string(packs.StatusDraft) {
		t.Fatalf("detail status = %v", detail["status"])
	}
	if stickers, ok := detail["stickers"].([]any); !ok || len(stickers) != 0 {
		t.Fatalf("draft detail stickers = %v", detail["stickers"])
	}
	if detail["bundle_url"] != "" {
		t.Fatalf("draft detail bundle_url = %v", detail["bundle_url"])
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/packs/"+pack.ID+"/process", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("process = %d: %s", rec.Code, rec.Body.String())
	}
	var job adminJob
	decodeInto(t, rec, &job)
	if job.Status != jobs.StatusQueued || job.PackID != pack.ID || job.Request.Rows != 1 {
		t.Fatalf("queued job = %+v", job)
	}
	rec = f.get("/api/v1/admin/jobs/" + job.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("get job = %d", rec.Code)
	}
	var fetched adminJob
	decodeInto(t, rec, &fetched)
	if fetched.ID != job.ID || fetched.Status != jobs.StatusQueued {
		t.Fatalf("fetched job = %+v", fetched)
	}

	// Draft delete is refused while a job is active, allowed without one.
	if rec := f.mutate(http.MethodDelete, "/api/v1/admin/packs/"+pack.ID, nil); rec.Code != http.StatusConflict {
		t.Fatalf("delete busy draft = %d, want 409", rec.Code)
	} else if code := errorCode(t, rec); code != "job_active" {
		t.Fatalf("delete busy draft code = %q", code)
	}
	rec = f.mutate(http.MethodPost, "/api/v1/admin/packs", map[string]any{"slug": "throwaway", "name": "Throwaway", "author": "Tester"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create throwaway = %d", rec.Code)
	}
	var throwaway adminPack
	decodeInto(t, rec, &throwaway)
	if rec := f.mutate(http.MethodDelete, "/api/v1/admin/packs/"+throwaway.ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete draft = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := f.get("/api/v1/admin/packs/" + throwaway.ID); rec.Code != http.StatusNotFound {
		t.Fatalf("deleted pack = %d, want 404", rec.Code)
	}

	// Revoke is only for committed packs and is idempotent.
	ready := f.readyPack("ready-pack")
	if rec := f.mutate(http.MethodDelete, "/api/v1/admin/packs/"+ready.ID, nil); rec.Code != http.StatusConflict {
		t.Fatalf("delete ready pack = %d, want 409", rec.Code)
	}
	rec = f.mutate(http.MethodPost, "/api/v1/admin/packs/"+ready.ID+"/revoke", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke = %d: %s", rec.Code, rec.Body.String())
	}
	var revoked adminPack
	decodeInto(t, rec, &revoked)
	if revoked.Status != packs.StatusRevoked {
		t.Fatalf("revoked status = %q", revoked.Status)
	}
	if rec := f.mutate(http.MethodPost, "/api/v1/admin/packs/"+ready.ID+"/revoke", nil); rec.Code != http.StatusOK {
		t.Fatalf("second revoke = %d", rec.Code)
	}
	tombstones, err := f.packs.Tombstones(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Tombstones: %v", err)
	}
	if len(tombstones) != 1 || tombstones[0].PackID != ready.ID {
		t.Fatalf("tombstones = %+v", tombstones)
	}
	key, err := f.auth.CreateKey(ctx, "sdk", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if rec := f.request(http.MethodGet, "/api/v1/packs/"+ready.ID, nil, map[string]string{"X-Api-Key": key.Raw}); rec.Code != http.StatusGone {
		t.Fatalf("SDK view of revoked pack = %d, want 410", rec.Code)
	}

	// Keys: list metadata, create, rotate, disable.
	rec = f.get("/api/v1/admin/api-keys")
	if rec.Code != http.StatusOK {
		t.Fatalf("list keys = %d", rec.Code)
	}
	var keyPage struct {
		Items []auth.Key `json:"items"`
	}
	decodeInto(t, rec, &keyPage)
	if len(keyPage.Items) != 1 {
		t.Fatalf("initial keys = %d, want the lifecycle key", len(keyPage.Items))
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/api-keys", map[string]any{
		"name": "studio", "expires_at": "2027-01-01T00:00:00Z",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create key = %d: %s", rec.Code, rec.Body.String())
	}
	var created auth.CreatedKey
	decodeInto(t, rec, &created)
	if created.Raw == "" || created.Prefix == "" {
		t.Fatalf("created key = %+v", created)
	}
	if rec := f.request(http.MethodGet, "/api/v1/packs", nil, map[string]string{"X-Api-Key": created.Raw}); rec.Code != http.StatusOK {
		t.Fatalf("created key SDK read = %d", rec.Code)
	}

	rec = f.get("/api/v1/admin/api-keys")
	var listed struct {
		Items []auth.Key `json:"items"`
	}
	decodeInto(t, rec, &listed)
	if strings.Contains(rec.Body.String(), created.Raw) || strings.Contains(rec.Body.String(), "key_hash") {
		t.Fatal("key listing leaked secret material")
	}
	var existing auth.Key
	for _, item := range listed.Items {
		if item.ID == created.ID {
			existing = item
		}
	}
	if !existing.Active || existing.Prefix != created.Prefix || existing.ExpiresAt == nil ||
		!existing.ExpiresAt.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("listed key = %+v", existing)
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/api-keys/"+created.ID+"/rotate", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("rotate = %d: %s", rec.Code, rec.Body.String())
	}
	var rotated auth.CreatedKey
	decodeInto(t, rec, &rotated)
	if rotated.ID == created.ID || rotated.Raw == created.Raw {
		t.Fatalf("rotated key = %+v", rotated)
	}
	if rec := f.request(http.MethodGet, "/api/v1/packs", nil, map[string]string{"X-Api-Key": created.Raw}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("rotated-away key SDK read = %d, want 401", rec.Code)
	}

	if rec := f.mutate(http.MethodDelete, "/api/v1/admin/api-keys/"+rotated.ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("disable = %d", rec.Code)
	}
	if rec := f.request(http.MethodGet, "/api/v1/packs", nil, map[string]string{"X-Api-Key": rotated.Raw}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled key SDK read = %d, want 401", rec.Code)
	}
	rec = f.get("/api/v1/admin/api-keys")
	listed.Items = nil
	decodeInto(t, rec, &listed)
	for _, item := range listed.Items {
		if item.ID == rotated.ID && item.Active {
			t.Fatalf("disabled key still active: %+v", item)
		}
	}
}
