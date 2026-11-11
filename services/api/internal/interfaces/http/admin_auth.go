package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/studio"
)

const (
	// maxJSONBody bounds every admin JSON request (spec §9.2).
	maxJSONBody = 1 << 20
	// maxMultipartBody bounds one upload: 8 MiB of file plus form overhead.
	maxMultipartBody = 8<<20 + 64<<10
	// maxUploadFile is the largest accepted image part.
	maxUploadFile = 8 << 20
	// loginMaxFailures is the failed-login budget per source per window.
	loginMaxFailures = 5
	loginWindow      = 15 * time.Minute
	// loginLimiterSources caps the memorized buckets before an expired sweep.
	loginLimiterSources = 1024
)

// admin owns the single-owner management API. Every route except login and
// logout requires a valid session cookie; unsafe requests must also carry the
// configured public origin. Client API keys never authenticate any admin route.
type admin struct {
	auth    *auth.Service
	studio  *studio.Service
	packs   *packs.Service
	jobs    *jobs.Runner
	assets  *assets.Store
	baseURL string
	trusted []netip.Prefix
	limiter *loginLimiter
	now     func() time.Time
}

func (a *admin) mount(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/admin/session", a.login)
	mux.HandleFunc("GET /api/v1/admin/session", a.sessionOnly(a.sessionInfo))
	mux.HandleFunc("DELETE /api/v1/admin/session", a.ownerOnly(a.logout))

	mux.HandleFunc("GET /api/v1/admin/fragment-groups", a.sessionOnly(a.listGroups))
	mux.HandleFunc("POST /api/v1/admin/fragment-groups", a.ownerOnly(a.createGroup))
	mux.HandleFunc("PUT /api/v1/admin/fragment-groups/{id}", a.ownerOnly(a.updateGroup))
	mux.HandleFunc("DELETE /api/v1/admin/fragment-groups/{id}", a.ownerOnly(a.deleteGroup))

	mux.HandleFunc("GET /api/v1/admin/fragments", a.sessionOnly(a.listFragments))
	mux.HandleFunc("POST /api/v1/admin/fragments", a.ownerOnly(a.createFragment))
	mux.HandleFunc("PUT /api/v1/admin/fragments/{id}", a.ownerOnly(a.updateFragment))
	mux.HandleFunc("DELETE /api/v1/admin/fragments/{id}", a.ownerOnly(a.archiveFragment))

	mux.HandleFunc("GET /api/v1/admin/characters", a.sessionOnly(a.listCharacters))
	mux.HandleFunc("POST /api/v1/admin/characters", a.ownerOnly(a.createCharacter))
	mux.HandleFunc("GET /api/v1/admin/characters/{id}", a.sessionOnly(a.getCharacter))
	mux.HandleFunc("PUT /api/v1/admin/characters/{id}", a.ownerOnly(a.updateCharacter))
	mux.HandleFunc("DELETE /api/v1/admin/characters/{id}", a.ownerOnly(a.archiveCharacter))

	mux.HandleFunc("POST /api/v1/admin/reference-assets", a.ownerOnly(a.uploadReferenceAsset))
	mux.HandleFunc("GET /api/v1/admin/reference-assets/{id}", a.sessionOnly(a.getReferenceAsset))

	mux.HandleFunc("GET /api/v1/admin/designs", a.sessionOnly(a.listDesigns))
	mux.HandleFunc("POST /api/v1/admin/designs", a.ownerOnly(a.saveDesign))
	mux.HandleFunc("POST /api/v1/admin/designs/compile", a.ownerOnly(a.compileDesign))
	mux.HandleFunc("POST /api/v1/admin/designs/suggest", a.ownerOnly(a.suggestDesign))
	mux.HandleFunc("GET /api/v1/admin/designs/{id}", a.sessionOnly(a.getDesign))
	mux.HandleFunc("PUT /api/v1/admin/designs/{id}", a.ownerOnly(a.updateDesign))
	mux.HandleFunc("DELETE /api/v1/admin/designs/{id}", a.ownerOnly(a.deleteDesign))
	mux.HandleFunc("GET /api/v1/admin/designs/{id}/sheets", a.sessionOnly(a.listSheets))
	mux.HandleFunc("POST /api/v1/admin/designs/{id}/sheets", a.ownerOnly(a.uploadSheet))
	mux.HandleFunc("GET /api/v1/admin/designs/{id}/sheets/{sheetId}/image", a.sessionOnly(a.getSheetImage))
	mux.HandleFunc("DELETE /api/v1/admin/designs/{id}/sheets/{sheetId}", a.ownerOnly(a.deleteSheet))

	mux.HandleFunc("GET /api/v1/admin/packs", a.sessionOnly(a.listPacks))
	mux.HandleFunc("POST /api/v1/admin/packs", a.ownerOnly(a.createPack))
	mux.HandleFunc("GET /api/v1/admin/packs/{id}", a.sessionOnly(a.getPack))
	mux.HandleFunc("PUT /api/v1/admin/packs/{id}", a.ownerOnly(a.updatePack))
	mux.HandleFunc("DELETE /api/v1/admin/packs/{id}", a.ownerOnly(a.deletePack))
	mux.HandleFunc("POST /api/v1/admin/packs/{id}/process", a.ownerOnly(a.processPack))
	mux.HandleFunc("POST /api/v1/admin/packs/{id}/revoke", a.ownerOnly(a.revokePack))
	mux.HandleFunc("GET /api/v1/admin/jobs/{id}", a.sessionOnly(a.getJob))

	mux.HandleFunc("GET /api/v1/admin/api-keys", a.sessionOnly(a.listKeys))
	mux.HandleFunc("POST /api/v1/admin/api-keys", a.ownerOnly(a.createKey))
	mux.HandleFunc("POST /api/v1/admin/api-keys/{id}/rotate", a.ownerOnly(a.rotateKey))
	mux.HandleFunc("DELETE /api/v1/admin/api-keys/{id}", a.ownerOnly(a.disableKey))
}

// sessionOnly rejects anything without a valid owner session cookie. API keys
// are deliberately ignored here (spec §10).
func (a *admin) sessionOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(w, r) {
			return
		}
		next(w, r)
	}
}

// ownerOnly adds the Origin check required on every unsafe admin request.
func (a *admin) ownerOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(w, r) {
			return
		}
		if !a.sameOrigin(r) {
			publicError(w, http.StatusForbidden, "origin_not_allowed", "origin must match the configured public base URL")
			return
		}
		next(w, r)
	}
}

func (a *admin) authorized(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(adminCookieName)
	if err != nil || cookie.Value == "" {
		publicError(w, http.StatusUnauthorized, "unauthorized", "owner session required")
		return false
	}
	if err := a.auth.ValidateSession(r.Context(), cookie.Value, a.now()); err != nil {
		publicError(w, http.StatusUnauthorized, "unauthorized", "owner session required")
		return false
	}
	return true
}

func (a *admin) sameOrigin(r *http.Request) bool {
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	return origin != "" && origin == a.baseURL
}

func (a *admin) login(w http.ResponseWriter, r *http.Request) {
	if !a.sameOrigin(r) {
		publicError(w, http.StatusForbidden, "origin_not_allowed", "origin must match the configured public base URL")
		return
	}
	source, err := auth.ClientSource(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), a.trusted)
	if err != nil {
		publicError(w, http.StatusBadRequest, "bad_request", "invalid client address")
		return
	}
	if !a.limiter.allowed(source) {
		publicError(w, http.StatusTooManyRequests, "rate_limited", "too many failed logins from this address")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	token, err := a.auth.Login(r.Context(), body.Password, a.now())
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		a.limiter.fail(source)
		publicError(w, http.StatusUnauthorized, "invalid_credentials", "invalid password")
		return
	case errors.Is(err, auth.ErrNoOwner):
		publicError(w, http.StatusConflict, "owner_missing", "owner is not initialised")
		return
	case err != nil:
		publicError(w, http.StatusInternalServerError, "internal", "could not start session")
		return
	}
	a.limiter.reset(source)
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.SessionTTL / time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *admin) sessionInfo(w http.ResponseWriter, _ *http.Request) {
	publicJSON(w, http.StatusOK, map[string]string{"username": auth.OwnerUsername})
}

func (a *admin) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(adminCookieName); err == nil {
		if err := a.auth.Logout(r.Context(), cookie.Value); err != nil {
			publicError(w, http.StatusInternalServerError, "internal", "could not revoke session")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// decodeJSON enforces the 1 MiB JSON cap, the JSON content type and a single
// strict document before any handler sees the payload.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		publicError(w, http.StatusUnprocessableEntity, "invalid_content_type", "content type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if tooLarge(err) {
			publicError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds 1 MiB")
			return false
		}
		publicError(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		publicError(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return false
	}
	return true
}

// upload streams the single "file" part of a bounded multipart body into save,
// which also receives every other text form field (read from the whole body, so
// field order does not matter). save may return its own status; 0 keeps the
// default. The handler writes the response because the part must be consumed first.
// upload accepts one image upload. The complete request body is spooled to a
// temp file under the total 8 MiB + 64 KiB bound and validated end to end before
// save runs, so trailing junk or a truncated tail can never mutate domain state.
func (a *admin) upload(w http.ResponseWriter, r *http.Request, status int,
	save func(r io.Reader, filename string, fields map[string]string) (any, int, error), onError func(http.ResponseWriter, error)) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	boundary := params["boundary"]
	if err != nil || mediaType != "multipart/form-data" || boundary == "" {
		publicError(w, http.StatusUnprocessableEntity, "invalid_content_type", "content type must be multipart/form-data")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartBody)
	spool, err := os.CreateTemp("", "restick-upload-*")
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not buffer upload")
		return
	}
	defer func() {
		_ = spool.Close()
		_ = os.Remove(spool.Name())
	}()
	if _, err := io.Copy(spool, r.Body); err != nil {
		if tooLarge(err) {
			publicError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "upload exceeds 8 MiB plus form overhead")
		} else {
			publicError(w, http.StatusBadRequest, "bad_request", "invalid multipart body")
		}
		return
	}

	filename, fields, err := inspectUpload(spool, boundary)
	switch {
	case errors.Is(err, errUploadTooLarge):
		publicError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "image exceeds 8 MiB")
		return
	case errors.Is(err, errUploadMissingFile):
		publicError(w, http.StatusBadRequest, "bad_request", "missing file part")
		return
	case err != nil:
		publicError(w, http.StatusBadRequest, "bad_request", "invalid multipart body")
		return
	}

	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not buffer upload")
		return
	}
	reader := multipart.NewReader(spool, boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			publicError(w, http.StatusBadRequest, "bad_request", "invalid multipart body")
			return
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		payload, saved, err := save(io.LimitReader(part, maxUploadFile+1), filename, fields)
		_ = part.Close()
		if err != nil {
			onError(w, err)
			return
		}
		if saved != 0 {
			status = saved
		}
		publicJSON(w, status, payload)
		return
	}
	publicError(w, http.StatusBadRequest, "bad_request", "missing file part")
}

var (
	errUploadTooLarge    = errors.New("httpx: upload file too large")
	errUploadMissingFile = errors.New("httpx: upload has no file part")
)

// maxUploadField bounds one text form field kept by inspectUpload.
const maxUploadField = 4 << 10

// inspectUpload drains the whole spooled body once, bounding the file part and
// rejecting malformed tails, without writing any domain state. It returns the
// file name and the text fields (each truncated to maxUploadField).
func inspectUpload(spool *os.File, boundary string) (string, map[string]string, error) {
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		return "", nil, err
	}
	reader := multipart.NewReader(spool, boundary)
	filename, found := "", false
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", nil, err
		}
		formName, name := part.FormName(), part.FileName()
		if formName != "file" && name == "" {
			value, err := io.ReadAll(io.LimitReader(part, maxUploadField))
			if err == nil {
				_, err = io.Copy(io.Discard, part)
			}
			_ = part.Close()
			if err != nil {
				return "", nil, err
			}
			if _, dup := fields[formName]; !dup {
				fields[formName] = string(value)
			}
			continue
		}
		size, err := io.Copy(io.Discard, part)
		_ = part.Close()
		if err != nil {
			return "", nil, err
		}
		if formName != "file" || found {
			continue
		}
		if size > maxUploadFile {
			return "", nil, errUploadTooLarge
		}
		filename, found = name, true
	}
	if !found {
		return "", nil, errUploadMissingFile
	}
	return filename, fields, nil
}

func tooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

// writeUploadError maps image/store failures of the upload path.
func writeUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, assets.ErrImageTooLarge):
		publicError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "image exceeds 8 MiB")
	case errors.Is(err, assets.ErrInvalidImage), errors.Is(err, assets.ErrImageDimensions),
		errors.Is(err, assets.ErrAnimatedImage), errors.Is(err, assets.ErrInvalidFilename):
		publicError(w, http.StatusUnprocessableEntity, "invalid_image", "file must be a still PNG, JPEG or WebP within 4096 px")
	case errors.Is(err, studio.ErrNotFound), errors.Is(err, assets.ErrNotFound):
		publicError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, studio.ErrFilesUnavailable):
		publicError(w, http.StatusInternalServerError, "internal", "file storage unavailable")
	default:
		publicError(w, http.StatusInternalServerError, "internal", "could not store upload")
	}
}

func writeStudioError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, studio.ErrNotFound):
		publicError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, studio.ErrGroupNotEmpty):
		publicError(w, http.StatusConflict, "group_not_empty", "group still has fragments")
	case errors.Is(err, studio.ErrSheetLimit):
		publicError(w, http.StatusConflict, "sheet_limit", "design already has the maximum number of sheets")
	case errors.Is(err, studio.ErrSheetInUse):
		publicError(w, http.StatusConflict, "sheet_in_use", "sheet is used by a pack")
	case errors.Is(err, studio.ErrDesignHasSheets):
		publicError(w, http.StatusConflict, "design_has_sheets", "delete the design's sheets first")
	case errors.Is(err, studio.ErrInvalid):
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "invalid request")
	default:
		publicError(w, http.StatusInternalServerError, "internal", "could not complete request")
	}
}

func writePackError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, packs.ErrNotFound), errors.Is(err, packs.ErrVersionNotFound):
		publicError(w, http.StatusNotFound, "not_found", "pack not found")
	case errors.Is(err, packs.ErrSlugTaken):
		publicError(w, http.StatusConflict, "slug_taken", "slug is already reserved")
	case errors.Is(err, packs.ErrJobActive), errors.Is(err, jobs.ErrJobActive):
		publicError(w, http.StatusConflict, "job_active", "pack has a queued or running job")
	case errors.Is(err, packs.ErrNotDraft):
		publicError(w, http.StatusConflict, "not_draft", "only a draft pack without a committed version can be deleted")
	case errors.Is(err, packs.ErrRevoked), errors.Is(err, jobs.ErrPackRevoked):
		publicError(w, http.StatusGone, "pack_revoked", "pack is revoked")
	case errors.Is(err, jobs.ErrNoSource):
		publicError(w, http.StatusUnprocessableEntity, "source_missing", "pack has no design sheet selected")
	case errors.Is(err, packs.ErrInvalid), errors.Is(err, jobs.ErrInvalid):
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "invalid request")
	case errors.Is(err, jobs.ErrNotFound):
		publicError(w, http.StatusNotFound, "not_found", "job not found")
	default:
		publicError(w, http.StatusInternalServerError, "internal", "could not complete request")
	}
}

func writeKeyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrKeyNotFound):
		publicError(w, http.StatusNotFound, "not_found", "api key not found")
	case errors.Is(err, auth.ErrEmptyKeyName):
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "key name must not be empty")
	default:
		publicError(w, http.StatusInternalServerError, "internal", "could not complete request")
	}
}

// loginLimiter counts failed logins per source address in a fixed window. The
// clock is injected so tests never sleep.
type loginLimiter struct {
	mu      sync.Mutex
	entries map[netip.Addr]loginBucket
	now     func() time.Time
}

type loginBucket struct {
	failures int
	expires  time.Time
}

func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{entries: map[netip.Addr]loginBucket{}, now: now}
}

func (l *loginLimiter) allowed(source netip.Addr) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, ok := l.entries[source]
	if !ok {
		return true
	}
	if !l.now().Before(bucket.expires) {
		delete(l.entries, source)
		return true
	}
	return bucket.failures < loginMaxFailures
}

func (l *loginLimiter) fail(source netip.Addr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.entries) >= loginLimiterSources {
		for addr, bucket := range l.entries {
			if !now.Before(bucket.expires) {
				delete(l.entries, addr)
			}
		}
	}
	bucket, ok := l.entries[source]
	if !ok || !now.Before(bucket.expires) {
		// shortcut: buckets are only swept above loginLimiterSources, upgrade to
		// a timer sweep if the map ever needs a hard memory bound.
		bucket = loginBucket{expires: now.Add(loginWindow)}
	}
	bucket.failures++
	l.entries[source] = bucket
}

func (l *loginLimiter) reset(source netip.Addr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, source)
}
