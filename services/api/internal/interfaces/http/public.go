package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/packs"
)

// adminCookieName is the owner session cookie set by the admin API.
const adminCookieName = "__Host-restick_admin"

// publicAPI serves the read-only endpoints of spec §9.1. Auth (client key or
// owner session) and pack visibility are always checked before any cache
// validation, range or file access.
type publicAPI struct {
	catalog *packs.Service
	files   *assets.Store
	auth    *auth.Service
	baseURL string
}

// NewPublicHandler returns the SDK-compatible read API. URLs come from the
// configured public origin, never from Host or forwarded headers.
func NewPublicHandler(catalog *packs.Service, files *assets.Store, authSvc *auth.Service, publicBaseURL string) http.Handler {
	api := &publicAPI{
		catalog: catalog,
		files:   files,
		auth:    authSvc,
		baseURL: strings.TrimRight(publicBaseURL, "/"),
	}
	mux := http.NewServeMux()
	api.mount(mux)
	return mux
}

// mount registers the SDK read routes on a shared mux so NewHandler can serve
// them beside the admin API and the static UI.
func (a *publicAPI) mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/packs", a.listPacks)
	mux.HandleFunc("GET /api/v1/packs/{id}", a.getPack)
	mux.HandleFunc("GET /api/v1/packs/{id}/bundle", a.bundleMeta)
	mux.HandleFunc("GET /api/v1/tombstones", a.tombstones)
	mux.HandleFunc("GET /v1/objects/{key...}", a.serveObject)
	mux.HandleFunc("GET /v1/bundles/{id}/v/{version}", a.serveBundle)
}

// authorize accepts a client key header or the owner session cookie. Query
// string keys are never consulted, so keys cannot leak into URLs or logs.
func (a *publicAPI) authorize(r *http.Request) bool {
	now := time.Now()
	if key := strings.TrimSpace(r.Header.Get("X-Api-Key")); key != "" {
		if err := a.auth.ValidateKey(r.Context(), key, now); err == nil {
			return true
		}
	}
	if cookie, err := r.Cookie(adminCookieName); err == nil {
		if err := a.auth.ValidateSession(r.Context(), cookie.Value, now); err == nil {
			return true
		}
	}
	return false
}

func publicJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func publicError(w http.ResponseWriter, status int, code, message string) {
	publicJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func (a *publicAPI) unauthorized(w http.ResponseWriter) {
	publicError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid credentials")
}

func (a *publicAPI) objectURL(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	return a.baseURL + "/v1/objects/" + strings.TrimPrefix(relativePath, "/")
}

func (a *publicAPI) summary(p packs.Pack) map[string]any {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{
		"id": p.ID, "slug": p.Slug, "name": p.Name, "author": p.Author, "tags": tags,
		"sticker_count": p.StickerCount, "version": p.CurrentVersion,
		"thumbnail_url": a.objectURL(p.ThumbnailPath), "checksum_sha256": p.ChecksumSHA256,
	}
}

func (a *publicAPI) listPacks(w http.ResponseWriter, r *http.Request) {
	if !a.authorize(r) {
		a.unauthorized(w)
		return
	}
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	size, _ := strconv.Atoi(query.Get("page_size"))
	result, err := a.catalog.ListReady(r.Context(), packs.Filter{
		Tag: query.Get("tag"), Page: page, PageSize: size,
	})
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not list packs")
		return
	}
	items := make([]map[string]any, 0, len(result.Items))
	for _, p := range result.Items {
		items = append(items, a.summary(p))
	}
	publicJSON(w, http.StatusOK, map[string]any{
		"items": items, "page": result.Page, "page_size": result.PageSize, "total": result.Total,
	})
}

// visiblePack resolves a pack for a public read: 404 for missing/draft,
// 410 for revoked.
func (a *publicAPI) visiblePack(w http.ResponseWriter, r *http.Request, id string) (packs.Pack, bool) {
	p, err := a.catalog.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, packs.ErrNotFound) {
			publicError(w, http.StatusNotFound, "not_found", "pack not found")
		} else {
			publicError(w, http.StatusInternalServerError, "internal", "could not load pack")
		}
		return packs.Pack{}, false
	}
	switch p.Status {
	case packs.StatusRevoked:
		publicError(w, http.StatusGone, "pack_revoked", "pack revoked")
		return packs.Pack{}, false
	case packs.StatusReady:
		return p, true
	default:
		publicError(w, http.StatusNotFound, "not_found", "pack not found")
		return packs.Pack{}, false
	}
}

func (a *publicAPI) getPack(w http.ResponseWriter, r *http.Request) {
	if !a.authorize(r) {
		a.unauthorized(w)
		return
	}
	p, ok := a.visiblePack(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	etag := `"sha256:` + p.ChecksumSHA256 + `"`
	w.Header().Set("ETag", etag)
	if r.Method == http.MethodHead {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		return
	}
	version, err := a.catalog.GetVersion(r.Context(), p.ID, p.CurrentVersion)
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not load pack version")
		return
	}
	stickers := make([]map[string]any, 0, len(version.Stickers))
	for _, sticker := range version.Stickers {
		keywords := sticker.Keywords
		if keywords == nil {
			keywords = []string{}
		}
		stickers = append(stickers, map[string]any{
			"file": sticker.Filename, "emoji_hint": sticker.EmojiHint, "keywords": keywords,
			"width": sticker.Width, "height": sticker.Height,
			"file_url": a.objectURL(sticker.ObjectPath), "thumb_url": a.objectURL(sticker.ThumbPath),
		})
	}
	payload := a.summary(p)
	payload["stickers"] = stickers
	publicJSON(w, http.StatusOK, payload)
}

func (a *publicAPI) bundleMeta(w http.ResponseWriter, r *http.Request) {
	if !a.authorize(r) {
		a.unauthorized(w)
		return
	}
	p, ok := a.visiblePack(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	publicJSON(w, http.StatusOK, map[string]any{
		"url":             a.baseURL + "/v1/bundles/" + p.ID + "/v/" + strconv.Itoa(p.CurrentVersion),
		"checksum_sha256": p.ChecksumSHA256,
		"version":         p.CurrentVersion,
	})
}

func (a *publicAPI) tombstones(w http.ResponseWriter, r *http.Request) {
	if !a.authorize(r) {
		a.unauthorized(w)
		return
	}
	since := time.Time{}
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			publicError(w, http.StatusBadRequest, "bad_request", "since must be an RFC3339 timestamp")
			return
		}
		since = parsed
	}
	items, err := a.catalog.Tombstones(r.Context(), since)
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not list tombstones")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"pack_id": item.PackID, "revoked_at": item.RevokedAt.UTC().Format("2006-01-02T15:04:05Z"),
			"reason_code": item.ReasonCode, "checksum_sha256": item.ChecksumSHA256,
		})
	}
	publicJSON(w, http.StatusOK, map[string]any{"items": out})
}
