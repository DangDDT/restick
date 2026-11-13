package httpx

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eykh/restick/api_svc/internal/animations"
	"github.com/eykh/restick/api_svc/internal/jobs"
)

// testClip is the smallest container the store sniffs as MP4.
var testClip = append([]byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, bytes.Repeat([]byte{7}, 64)...)

// animationPack commits a ready v1 pack with one 512 px sticker whose committed
// WebP is testdata/sticker.webp, so on-white and preview serve real files.
func (f *adminFixture) animationPack(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	pack, err := f.packs.Create(ctx, seedSheetDraft(t, f.db, f.files, "animated-pack", "Animated pack", []string{"demo"}))
	if err != nil {
		t.Fatalf("packs.Create: %v", err)
	}
	static, err := os.ReadFile(filepath.Join("testdata", "sticker.webp"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	outputDir := fmt.Sprintf("packs/%s/v1", pack.ID)
	objectPath := outputDir + "/000.webp"
	target := filepath.Join(f.root, filepath.FromSlash(objectPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, static, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`INSERT INTO pack_versions
		(pack_id, version, job_id, checksum_sha256, sticker_count, manifest_json, bundle_path, bundle_byte_size, thumbnail_path, output_dir)
		VALUES (?, 1, ?, ?, 1, '{}', ?, 10, ?, ?)`,
		pack.ID, "job-"+pack.ID, strings.Repeat("b", 64),
		fmt.Sprintf("bundles/%s/v1.zip", pack.ID), outputDir+"/thumb.webp", outputDir); err != nil {
		t.Fatalf("insert version: %v", err)
	}
	if _, err := f.db.Exec(`INSERT INTO stickers
		(pack_id, version, sort_order, filename, object_path, thumb_path, checksum_sha256, width, height, emoji_hint, keywords_json)
		VALUES (?, 1, 0, '000.webp', ?, ?, ?, 512, 512, '👋', '["wave"]')`,
		pack.ID, objectPath, outputDir+"/thumbs/thumb_000.webp", strings.Repeat("c", 64)); err != nil {
		t.Fatalf("insert sticker: %v", err)
	}
	if _, err := f.db.Exec(`UPDATE packs SET status = 'ready', current_version = 1 WHERE id = ?`, pack.ID); err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	return pack.ID
}

// uploadClip posts the clip part plus its motion text, the browser order.
func (f *adminFixture) uploadClip(t *testing.T, packID string, index int, motionText string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("clip", "clip.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(testClip); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("motion_text", motionText); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	headers := f.adminHeaders()
	headers["Content-Type"] = writer.FormDataContentType()
	return f.request(http.MethodPost, animationPath(packID, index), buf.Bytes(), headers)
}

func animationPath(packID string, index int) string {
	return fmt.Sprintf("/api/v1/admin/packs/%s/stickers/%d/animation", packID, index)
}

// markAnimationReady writes the worker's output row and file behind a draft.
func (f *adminFixture) markAnimationReady(t *testing.T, packID string) {
	t.Helper()
	var id string
	if err := f.db.QueryRow(`SELECT id FROM sticker_animations WHERE pack_id = ? AND sticker_index = 0`, packID).Scan(&id); err != nil {
		t.Fatalf("read draft: %v", err)
	}
	animatedPath := "animations/" + id + ".webp"
	target := filepath.Join(f.root, filepath.FromSlash(animatedPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("RIFF....WEBPVP8L"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE sticker_animations SET status = 'ready', animated_path = ?,
		animated_checksum_sha256 = ?, frames = 36, fps = 12, duration_ms = 3000, quality = 75,
		width = 512, height = 512, animated_byte_size = 16 WHERE id = ?`,
		animatedPath, strings.Repeat("a", 64), id); err != nil {
		t.Fatalf("mark ready: %v", err)
	}
}

func TestUploadAnimationAccepts202(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)

	rec := f.uploadClip(t, packID, 0, "  vẫy tay  ")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("upload = %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeMap(t, rec)
	if payload["status"] != "queued" || payload["sticker_index"] != float64(0) || payload["motion_text"] != "vẫy tay" {
		t.Fatalf("animation = %v", payload)
	}
	if payload["use_animated"] != true || payload["preview_url"] != "https://sticker.example/api/v1/admin/packs/"+packID+"/stickers/0/animation/preview" {
		t.Fatalf("animation defaults = %v", payload)
	}

	list := f.get("/api/v1/admin/packs/" + packID + "/animations")
	if list.Code != http.StatusOK {
		t.Fatalf("list = %d: %s", list.Code, list.Body.String())
	}
	if items, _ := decodeMap(t, list)["items"].([]any); len(items) != 1 {
		t.Fatalf("list items = %v", decodeMap(t, list)["items"])
	}

	detail := decodeMap(t, f.get("/api/v1/admin/packs/"+packID))
	stickers := detail["stickers"].([]any)
	summary, _ := stickers[0].(map[string]any)["animation"].(map[string]any)
	if summary["status"] != "queued" || summary["use_animated"] != true || summary["stale"] != false {
		t.Fatalf("pack detail animation = %v", stickers[0])
	}
	if _, committed := stickers[0].(map[string]any)["animated"]; committed {
		t.Fatalf("static sticker has committed animated = %v", stickers[0])
	}
}

func TestUploadRejectedWithoutOwnerSession(t *testing.T) {
	f := newAdminFixture(t)
	packID := f.animationPack(t)
	body, contentType := multipartBody(t, "clip", "clip.mp4", testClip)

	if rec := f.request(http.MethodPost, animationPath(packID, 0), body,
		map[string]string{"Content-Type": contentType, adminOriginKey: adminBaseURL}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("upload without session = %d: %s", rec.Code, rec.Body.String())
	}
	key, err := f.auth.CreateKey(context.Background(), "sdk", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	rec := f.request(http.MethodPost, animationPath(packID, 0), body,
		map[string]string{"X-Api-Key": key.Raw, "Content-Type": contentType, adminOriginKey: adminBaseURL})
	if rec.Code != http.StatusUnauthorized || errorCode(t, rec) != "unauthorized" {
		t.Fatalf("client key upload = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := f.request(http.MethodGet, "/api/v1/admin/packs/"+packID+"/animations", nil,
		map[string]string{"X-Api-Key": key.Raw}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("client key list = %d", rec.Code)
	}
}

func TestUploadRejectsOverlongMotionText(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	rec := f.uploadClip(t, packID, 0, strings.Repeat("ấ", 501))
	if rec.Code != http.StatusUnprocessableEntity || errorCode(t, rec) != "unprocessable" {
		t.Fatalf("overlong motion = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPreviewRequiresSession(t *testing.T) {
	f := newAdminFixture(t)
	packID := f.animationPack(t)
	rec := f.request(http.MethodGet, animationPath(packID, 0)+"/preview", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("preview without session = %d", rec.Code)
	}
}

func TestPreviewServesReadyDraftOnly(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	if rec := f.uploadClip(t, packID, 0, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("upload = %d: %s", rec.Code, rec.Body.String())
	}
	rec := f.get(animationPath(packID, 0) + "/preview")
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "not_ready" {
		t.Fatalf("queued preview = %d: %s", rec.Code, rec.Body.String())
	}
	f.markAnimationReady(t, packID)
	rec = f.get(animationPath(packID, 0) + "/preview")
	if rec.Code != http.StatusOK {
		t.Fatalf("ready preview = %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/webp" || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("preview headers = %v", rec.Header())
	}
	if !bytes.Equal(rec.Body.Bytes(), []byte("RIFF....WEBPVP8L")) {
		t.Fatalf("preview body = %q", rec.Body.Bytes())
	}
}

func TestDeleteAnimationDraft(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	if rec := f.uploadClip(t, packID, 0, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("upload = %d", rec.Code)
	}
	if rec := f.mutate(http.MethodDelete, animationPath(packID, 0), nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d: %s", rec.Code, rec.Body.String())
	}
	items, _ := decodeMap(t, f.get("/api/v1/admin/packs/"+packID+"/animations"))["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("drafts after delete = %v", items)
	}
}

func TestPublishNoChanges409(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	rec := f.mutate(http.MethodPost, "/api/v1/admin/packs/"+packID+"/publish-animations", nil)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "no_changes" {
		t.Fatalf("publish without animations = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompilePromptContainsMotionAndBackgroundRules(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	rec := f.mutate(http.MethodPost, "/api/v1/admin/animations/compile", map[string]any{
		"pack_id": packID, "sticker_index": 0, "motion_text": "waving one paw",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("compile = %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeMap(t, rec)
	prompt, _ := payload["prompt"].(string)
	if !strings.Contains(prompt, "waving one paw") || !strings.Contains(strings.ToLower(prompt), "white") {
		t.Fatalf("prompt = %q", prompt)
	}
	if payload["emoji_hint"] != "👋" {
		t.Fatalf("emoji_hint = %v", payload["emoji_hint"])
	}
	if rec := f.mutate(http.MethodPost, "/api/v1/admin/animations/compile", map[string]any{
		"pack_id": packID, "sticker_index": 7, "motion_text": "waving",
	}); rec.Code != http.StatusNotFound {
		t.Fatalf("compile unknown sticker = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOnWhiteIsOpaqueWhiteBackground(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	rec := f.get(fmt.Sprintf("/api/v1/admin/packs/%s/stickers/0/on-white.png", packID))
	if rec.Code != http.StatusOK {
		t.Fatalf("on-white = %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("on-white headers = %v", rec.Header())
	}
	img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	if img.Bounds() != image.Rect(0, 0, 512, 512) {
		t.Fatalf("on-white size = %v", img.Bounds())
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 0xffff || g != 0xffff || b != 0xffff || a != 0xffff {
		t.Fatalf("corner pixel = %d,%d,%d,%d, want opaque white", r>>8, g>>8, b>>8, a>>8)
	}
	if r, g, b, _ := img.At(256, 256).RGBA(); r>>8 != 250 || g>>8 != 210 || b>>8 != 40 {
		t.Fatalf("center pixel = %d,%d,%d, want the sticker colour", r>>8, g>>8, b>>8)
	}
}

func TestAnimationErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"animation job active", animations.ErrJobActive, http.StatusConflict, "job_active"},
		{"publish job active", jobs.ErrJobActive, http.StatusConflict, "job_active"},
		{"animation busy", jobs.ErrAnimationBusy, http.StatusConflict, "animation_busy"},
		{"no changes", jobs.ErrNoChanges, http.StatusConflict, "no_changes"},
		{"not ready", animations.ErrNotReady, http.StatusConflict, "not_ready"},
		{"invalid", animations.ErrInvalid, http.StatusUnprocessableEntity, "unprocessable"},
		{"not found", animations.ErrNotFound, http.StatusNotFound, "not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeAnimationError(rec, tc.err)
			if rec.Code != tc.status || errorCode(t, rec) != tc.code {
				t.Fatalf("%v = %d %s, want %d %s", tc.err, rec.Code, errorCode(t, rec), tc.status, tc.code)
			}
		})
	}
}

func TestAdminPackDetailShowsCommittedAnimated(t *testing.T) {
	f := newAdminFixture(t)
	f.login()
	packID := f.animationPack(t)
	if _, err := f.db.Exec(`UPDATE stickers SET animated_path = ?, animated_checksum_sha256 = ?,
		animated_byte_size = 1000, animated_frames = 36, animated_duration_ms = 3000
		WHERE pack_id = ? AND version = 1 AND sort_order = 0`,
		fmt.Sprintf("packs/%s/v1/animated/000.webp", packID), strings.Repeat("d", 64), packID); err != nil {
		t.Fatal(err)
	}
	stickers := decodeMap(t, f.get("/api/v1/admin/packs/"+packID))["stickers"].([]any)
	animated, _ := stickers[0].(map[string]any)["animated"].(map[string]any)
	if animated["checksum_sha256"] != strings.Repeat("d", 64) || animated["frames"] != float64(36) || animated["duration_ms"] != float64(3000) {
		t.Fatalf("committed animated = %v", stickers[0])
	}
	if animated["object_url"] != fmt.Sprintf("https://sticker.example/v1/objects/packs/%s/v1/animated/000.webp", packID) {
		t.Fatalf("animated object_url = %v", animated["object_url"])
	}
}
