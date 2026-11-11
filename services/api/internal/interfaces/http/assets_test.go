package httpx

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func objectURLFor(rel string) string {
	return testPublicBase + "/v1/objects/" + rel
}

func TestBundleChecksumContract(t *testing.T) {
	f := newPublicFixture(t)
	created := f.publish("alpha", 2)
	id := created.pack.ID
	f.grant(f.keyID, id)
	meta := decodeBody(t, f.request(http.MethodGet, "/api/v1/packs/"+id+"/bundle", f.keyHeaders()))

	rec := f.request(http.MethodGet, "/v1/bundles/"+id+"/v/1", f.keyHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("bundle status = %d body=%s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("bundle content type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "private, no-cache" {
		t.Fatalf("bundle cache control = %q", cc)
	}
	if etag := rec.Header().Get("ETag"); etag != `"sha256:`+created.checksum+`"` {
		t.Fatalf("bundle etag = %q", etag)
	}
	if !bytes.Equal(rec.Body.Bytes(), created.bundle) {
		t.Fatal("bundle bytes differ from the committed ZIP")
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("open served ZIP: %v", err)
	}
	manifestFile, err := reader.Open("pack.json")
	if err != nil {
		t.Fatalf("ZIP pack.json: %v", err)
	}
	manifestJSON, err := io.ReadAll(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Schema         string `json:"schema"`
		ChecksumSHA256 string `json:"checksum_sha256"`
		Stickers       []struct {
			File  string `json:"file"`
			Thumb string `json:"thumb"`
		} `json:"stickers"`
	}
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "restick.pack.v1" {
		t.Fatalf("manifest schema = %q", manifest.Schema)
	}
	if manifest.ChecksumSHA256 != created.checksum || meta["checksum_sha256"] != created.checksum {
		t.Fatalf("checksum manifest=%q meta=%v", manifest.ChecksumSHA256, meta["checksum_sha256"])
	}

	// The SDK checksum is SHA256 over sticker bytes concatenated in manifest
	// order, not over the ZIP.
	hasher := sha256.New()
	for _, sticker := range manifest.Stickers {
		object := f.request(http.MethodGet, objectURLFor("packs/"+id+"/v1/"+sticker.File), f.keyHeaders())
		if object.Code != http.StatusOK {
			t.Fatalf("object %s status = %d", sticker.File, object.Code)
		}
		if object.Header().Get("Content-Type") != "image/webp" {
			t.Fatalf("object content type = %q", object.Header().Get("Content-Type"))
		}
		hasher.Write(object.Body.Bytes())
		if _, err := reader.Open(sticker.Thumb); err != nil {
			t.Fatalf("manifest thumb %s missing from ZIP: %v", sticker.Thumb, err)
		}
	}
	if actual := hex.EncodeToString(hasher.Sum(nil)); actual != created.checksum {
		t.Fatalf("manifest-order checksum = %s, want %s", actual, created.checksum)
	}
}

func TestOldCommittedVersionStillDownloads(t *testing.T) {
	f := newPublicFixture(t)
	v1 := f.publish("alpha", 2)
	v2 := f.publishNextVersion(v1, 2)
	id := v1.pack.ID
	f.grant(f.keyID, id)

	old := f.request(http.MethodGet, "/v1/bundles/"+id+"/v/1", f.keyHeaders())
	if old.Code != http.StatusOK || !bytes.Equal(old.Body.Bytes(), v1.bundle) {
		t.Fatalf("old bundle status=%d match=%v", old.Code, bytes.Equal(old.Body.Bytes(), v1.bundle))
	}
	current := f.request(http.MethodGet, "/v1/bundles/"+id+"/v/2", f.keyHeaders())
	if current.Code != http.StatusOK || !bytes.Equal(current.Body.Bytes(), v2.bundle) {
		t.Fatalf("current bundle status=%d", current.Code)
	}
	oldObject := f.request(http.MethodGet, objectURLFor("packs/"+id+"/v1/000.webp"), f.keyHeaders())
	if oldObject.Code != http.StatusOK || !bytes.Equal(oldObject.Body.Bytes(), v1.files["packs/"+id+"/v1/000.webp"]) {
		t.Fatalf("old object status=%d", oldObject.Code)
	}

	f.revoke(id, "2026-11-10T10:00:00Z")
	for _, target := range []string{
		"/v1/bundles/" + id + "/v/1",
		"/v1/bundles/" + id + "/v/2",
		objectURLFor("packs/" + id + "/v1/000.webp"),
	} {
		rec := f.request(http.MethodGet, target, f.keyHeaders())
		if rec.Code != http.StatusGone {
			t.Fatalf("revoked %s status = %d", target, rec.Code)
		}
		if code := decodeBody(t, rec)["error"].(map[string]any)["code"]; code != "pack_revoked" {
			t.Fatalf("revoked %s code = %v", target, code)
		}
	}
}

func TestAuthPrecedesHeadRangeAnd304(t *testing.T) {
	f := newPublicFixture(t)
	created := f.publish("alpha", 2)
	id := created.pack.ID
	f.grant(f.keyID, id)
	bundleURL := "/v1/bundles/" + id + "/v/1"
	object := objectURLFor("packs/" + id + "/v1/000.webp")

	ok := f.request(http.MethodGet, bundleURL, f.keyHeaders())
	if ok.Code != http.StatusOK {
		t.Fatalf("bundle status = %d", ok.Code)
	}
	etag := ok.Header().Get("ETag")
	if etag == "" || ok.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("etag=%q accept-ranges=%q", etag, ok.Header().Get("Accept-Ranges"))
	}

	notModified := f.request(http.MethodGet, bundleURL, map[string]string{"X-Api-Key": f.apiKey, "If-None-Match": etag})
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d", notModified.Code)
	}

	anonymous304 := f.request(http.MethodGet, bundleURL, map[string]string{"If-None-Match": etag})
	if anonymous304.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous conditional status = %d", anonymous304.Code)
	}
	if head := f.request(http.MethodHead, bundleURL, nil); head.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous HEAD status = %d", head.Code)
	}
	if head := f.request(http.MethodHead, object, f.keyHeaders()); head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD status=%d body=%d", head.Code, head.Body.Len())
	}
	if ranged := f.request(http.MethodGet, bundleURL, map[string]string{"Range": "bytes=0-3"}); ranged.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous range status = %d", ranged.Code)
	}
	ranged := f.request(http.MethodGet, bundleURL, map[string]string{"X-Api-Key": f.apiKey, "Range": "bytes=0-3"})
	if ranged.Code != http.StatusPartialContent || ranged.Body.Len() != 4 ||
		!strings.HasPrefix(ranged.Header().Get("Content-Range"), "bytes 0-3/") {
		t.Fatalf("range status=%d len=%d content-range=%q", ranged.Code, ranged.Body.Len(), ranged.Header().Get("Content-Range"))
	}
	objectOK := f.request(http.MethodGet, object, f.keyHeaders())
	if objectOK.Code != http.StatusOK || objectOK.Header().Get("ETag") == "" {
		t.Fatalf("object status=%d etag=%q", objectOK.Code, objectOK.Header().Get("ETag"))
	}
	objectNotModified := f.request(http.MethodGet, object, map[string]string{"X-Api-Key": f.apiKey, "If-None-Match": objectOK.Header().Get("ETag")})
	if objectNotModified.Code != http.StatusNotModified {
		t.Fatalf("object conditional status = %d", objectNotModified.Code)
	}

	f.revoke(id, "2026-11-10T10:00:00Z")
	revoked := f.request(http.MethodGet, bundleURL, map[string]string{"X-Api-Key": f.apiKey, "If-None-Match": etag})
	if revoked.Code != http.StatusGone {
		t.Fatalf("revoked conditional status = %d", revoked.Code)
	}
}

func TestAdminCookieLoadsThumbnail(t *testing.T) {
	f := newPublicFixture(t)
	created := f.publish("alpha", 1)
	id := created.pack.ID
	thumbnail := objectURLFor("packs/" + id + "/v1/thumb.webp")
	cookie := map[string]string{"Cookie": adminCookieName + "=" + f.session}

	rec := f.request(http.MethodGet, thumbnail, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("cookie thumbnail status = %d body=%s", rec.Code, rec.Body)
	}
	if rec.Body.String() != string(created.files["packs/"+id+"/v1/thumb.webp"]) {
		t.Fatal("cookie thumbnail bytes differ")
	}
	if rec.Header().Get("Content-Type") != "image/webp" || rec.Header().Get("Cache-Control") != "private, no-cache" {
		t.Fatalf("thumbnail headers = %v", rec.Header())
	}
	files := f.request(http.MethodGet, objectURLFor("packs/"+id+"/v1/000.webp"), cookie)
	if files.Code != http.StatusOK {
		t.Fatalf("cookie sticker status = %d", files.Code)
	}
	catalog := f.request(http.MethodGet, "/api/v1/packs", cookie)
	if catalog.Code != http.StatusOK {
		t.Fatalf("cookie catalog status = %d", catalog.Code)
	}
	fake := f.request(http.MethodGet, thumbnail, map[string]string{"Cookie": adminCookieName + "=not-a-session"})
	if fake.Code != http.StatusUnauthorized {
		t.Fatalf("fake cookie status = %d", fake.Code)
	}
}

func TestQueryAPIKeyDoesNotAuthorize(t *testing.T) {
	f := newPublicFixture(t)
	created := f.publish("alpha", 1)
	id := created.pack.ID
	f.grant(f.keyID, id)
	for _, target := range []string{
		"/api/v1/packs?api_key=" + f.apiKey,
		"/api/v1/packs/" + id + "?key=" + f.apiKey,
		"/api/v1/packs/" + id + "/bundle?api_key=" + f.apiKey,
		"/api/v1/tombstones?api_key=" + f.apiKey,
		objectURLFor("packs/"+id+"/v1/000.webp") + "?api_key=" + f.apiKey,
		"/v1/bundles/" + id + "/v/1?api_key=" + f.apiKey,
	} {
		rec := f.request(http.MethodGet, target, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("query key %s status = %d", target, rec.Code)
		}
	}
}
