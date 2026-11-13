package httpx

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/packs"
)

// serveObject streams one file of a committed, ready pack version. The key is
// a data-root relative path that must match a row in the version allowlist;
// there is no generic file browser.
func (a *publicAPI) serveObject(w http.ResponseWriter, r *http.Request) {
	who, ok := a.authenticate(r)
	if !ok {
		a.unauthorized(w)
		return
	}
	rel := strings.TrimPrefix(r.PathValue("key"), "/")
	packID, version, ok := splitObjectKey(rel)
	if !ok || !a.granted(r, who, packID) {
		publicError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	p, err := a.catalog.Get(r.Context(), packID)
	if err != nil {
		publicError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	if p.Status == packs.StatusRevoked {
		publicError(w, http.StatusGone, "pack_revoked", "pack revoked")
		return
	}
	if p.Status != packs.StatusReady {
		publicError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	v, err := a.catalog.GetVersion(r.Context(), packID, version)
	if err != nil {
		if errors.Is(err, packs.ErrVersionNotFound) || errors.Is(err, packs.ErrNotFound) {
			publicError(w, http.StatusNotFound, "not_found", "file not found")
		} else {
			publicError(w, http.StatusInternalServerError, "internal", "could not load pack version")
		}
		return
	}
	checksum, ok := versionObjectChecksum(v, rel)
	if !ok {
		publicError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	file, err := a.files.OpenCommitted(rel)
	if err != nil {
		publicError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	defer file.Close()
	serveCommittedFile(w, r, rel, checksum, v.CreatedAt, file)
}

// serveBundle streams the ZIP of any committed version of a ready pack. A
// revoked pack blocks every version.
func (a *publicAPI) serveBundle(w http.ResponseWriter, r *http.Request) {
	who, ok := a.authenticate(r)
	if !ok {
		a.unauthorized(w)
		return
	}
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		publicError(w, http.StatusNotFound, "not_found", "bundle not found")
		return
	}
	packID := r.PathValue("id")
	if !a.granted(r, who, packID) {
		publicError(w, http.StatusNotFound, "not_found", "bundle not found")
		return
	}
	p, err := a.catalog.Get(r.Context(), packID)
	if err != nil {
		publicError(w, http.StatusNotFound, "not_found", "bundle not found")
		return
	}
	if p.Status == packs.StatusRevoked {
		publicError(w, http.StatusGone, "pack_revoked", "pack revoked")
		return
	}
	if p.Status != packs.StatusReady {
		publicError(w, http.StatusNotFound, "not_found", "bundle not found")
		return
	}
	v, err := a.catalog.GetVersion(r.Context(), packID, version)
	if err != nil {
		if errors.Is(err, packs.ErrVersionNotFound) || errors.Is(err, packs.ErrNotFound) {
			publicError(w, http.StatusNotFound, "not_found", "bundle not found")
		} else {
			publicError(w, http.StatusInternalServerError, "internal", "could not load pack version")
		}
		return
	}
	file, err := a.files.OpenCommitted(v.BundlePath)
	if err != nil {
		publicError(w, http.StatusNotFound, "not_found", "bundle not found")
		return
	}
	defer file.Close()
	serveCommittedFile(w, r, v.BundlePath, v.ChecksumSHA256, v.CreatedAt, file)
}

// versionObjectChecksum returns the immutable checksum of one allowlisted
// member of a committed version.
func versionObjectChecksum(v packs.Version, rel string) (string, bool) {
	if rel == v.ThumbnailPath {
		if len(v.Stickers) > 0 {
			return v.Stickers[0].ChecksumSHA256, true
		}
		return v.ChecksumSHA256, true
	}
	if v.OutputDir != "" && rel == path.Join(v.OutputDir, "pack.json") {
		return v.ChecksumSHA256, true
	}
	for _, sticker := range v.Stickers {
		if rel == sticker.ObjectPath || rel == sticker.ThumbPath {
			return sticker.ChecksumSHA256, true
		}
		if sticker.AnimatedPath != "" && rel == sticker.AnimatedPath {
			return sticker.AnimatedChecksumSHA256, true
		}
	}
	return "", false
}

// serveCommittedFile serves immutable version bytes with range support. The
// Cache-Control keeps every response revalidated through auth so a shared
// cache can never bypass key or revocation checks.
func serveCommittedFile(w http.ResponseWriter, r *http.Request, rel, checksum string, modified time.Time, file *os.File) {
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("Content-Type", contentTypeFor(rel))
	if checksum != "" {
		w.Header().Set("ETag", `"sha256:`+checksum+`"`)
	}
	http.ServeContent(w, r, path.Base(rel), modified, file)
}

// splitObjectKey parses "packs/<packID>/v<version>/<member...>". The UUID and
// version are re-validated by the catalog; malformed keys are a 404.
func splitObjectKey(rel string) (string, int, bool) {
	parts := strings.Split(rel, "/")
	if len(parts) < 4 || parts[0] != "packs" || !strings.HasPrefix(parts[2], "v") {
		return "", 0, false
	}
	version, err := strconv.Atoi(strings.TrimPrefix(parts[2], "v"))
	if err != nil || version < 1 || parts[3] == "" {
		return "", 0, false
	}
	return parts[1], version, true
}

func contentTypeFor(rel string) string {
	switch path.Ext(rel) {
	case ".webp":
		return "image/webp"
	case ".json":
		return "application/json"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
