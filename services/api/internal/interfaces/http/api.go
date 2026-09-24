package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/domain/apikey"
	"github.com/eykh/restick/api_svc/internal/domain/pack"
	"github.com/eykh/restick/api_svc/internal/domain/user"
	"github.com/eykh/restick/api_svc/internal/infrastructure/postgres"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	store         *postgres.Store
	jwtSecret     []byte
	webOrigins    []string
	publicFTP     string
}

func NewServer(store *postgres.Store, jwtSecret string, webOrigins []string, publicFTP string) *Server {
	return &Server{store: store, jwtSecret: []byte(jwtSecret), webOrigins: webOrigins, publicFTP: strings.TrimRight(publicFTP, "/")}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /docs", s.swaggerUI)
	mux.HandleFunc("GET /docs/", s.swaggerUI)
	mux.HandleFunc("GET /openapi.yaml", s.openAPI)
	mux.HandleFunc("POST /api/v1/auth/register", s.register)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/me", s.me)
	mux.HandleFunc("GET /api/v1/packs", s.listPublished)
	mux.HandleFunc("GET /api/v1/packs/{id}", s.getPack)
	mux.HandleFunc("HEAD /api/v1/packs/{id}", s.headPack)
	mux.HandleFunc("GET /api/v1/packs/{id}/bundle", s.bundleMeta)
	mux.HandleFunc("GET /api/v1/tombstones", s.tombstones)
	mux.HandleFunc("POST /api/v1/packs", s.createPack)
	mux.HandleFunc("PUT /api/v1/packs/{id}", s.updatePack)
	mux.HandleFunc("PUT /api/v1/packs/{id}/stickers", s.updateStickersMetadata)
	mux.HandleFunc("GET /api/v1/me/packs", s.myPacks)
	mux.HandleFunc("POST /api/v1/packs/{id}/submit", s.submitPack)
	mux.HandleFunc("POST /api/v1/packs/{id}/confirm", s.confirmPack)
	mux.HandleFunc("DELETE /api/v1/packs/{id}", s.deleteRejectedPack)
	mux.HandleFunc("POST /api/v1/packs/{id}/revoke", s.ownerRevokePack)
	mux.HandleFunc("GET /api/v1/admin/packs", s.adminQueue)
	mux.HandleFunc("GET /api/v1/admin/packs/{id}", s.adminGet)
	mux.HandleFunc("POST /api/v1/admin/packs/{id}/approve", s.approve)
	mux.HandleFunc("POST /api/v1/admin/packs/{id}/reject", s.reject)
	mux.HandleFunc("POST /api/v1/admin/packs/{id}/revoke", s.revoke)
	mux.HandleFunc("GET /api/v1/admin/published", s.adminPublished)
	mux.HandleFunc("GET /api/v1/admin/analytics/rejections", s.rejectionAnalytics)
	mux.HandleFunc("GET /api/v1/me/api-keys", s.listMyAPIKeys)
	mux.HandleFunc("POST /api/v1/me/api-keys", s.createMyAPIKey)
	mux.HandleFunc("POST /api/v1/me/api-keys/{id}/rotate", s.rotateMyAPIKey)
	mux.HandleFunc("DELETE /api/v1/me/api-keys/{id}", s.deleteMyAPIKey)
	mux.HandleFunc("GET /api/v1/admin/api-keys", s.adminListAPIKeys)
	mux.HandleFunc("DELETE /api/v1/admin/api-keys/{id}", s.adminDeleteAPIKey)
	return s.cors(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api_svc"})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if len(body.Password) < 8 || !strings.Contains(body.Email, "@") {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid email or password")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "hash failed")
		return
	}
	name := body.DisplayName
	if name == "" {
		name = strings.Split(body.Email, "@")[0]
	}
	u, err := s.store.CreateUser(r.Context(), strings.ToLower(strings.TrimSpace(body.Email)), string(hash), name, user.RoleContributor)
	if err != nil {
		if errors.Is(err, user.ErrEmailTaken) {
			writeErr(w, http.StatusConflict, "conflict", "email taken")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	s.writeToken(w, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	u, err := s.store.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(body.Email)))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}
	s.writeToken(w, u)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, userJSON(u))
}

func (s *Server) listPublished(w http.ResponseWriter, r *http.Request) {
	if !s.mustAPIKey(w, r) {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}
	items, total, err := s.store.ListPublished(r.Context(), page, size, r.URL.Query().Get("tag"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, s.summaryJSON(r, p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "page": page, "page_size": size, "total": total})
}

func (s *Server) getPack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.store.GetPack(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if u, err := s.userFromRequest(r); err == nil {
		if u.ID == p.SubmitterID || u.IsAdmin() {
			s.writeDetail(w, r, p)
			return
		}
	}
	if !s.apiKeyOK(r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	if p.Status == pack.StatusRevoked {
		writeErr(w, http.StatusGone, "pack_revoked", "pack revoked")
		return
	}
	if p.Status != pack.StatusPublished {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	s.writeDetail(w, r, p)
}

func (s *Server) headPack(w http.ResponseWriter, r *http.Request) {
	if !s.mustAPIKey(w, r) {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if p.Status == pack.StatusRevoked {
		w.WriteHeader(http.StatusGone)
		return
	}
	if p.Status != pack.StatusPublished {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("ETag", `"sha256:`+p.ChecksumSHA256+`"`)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) bundleMeta(w http.ResponseWriter, r *http.Request) {
	if !s.mustAPIKey(w, r) {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if p.Status == pack.StatusRevoked {
		writeErr(w, http.StatusGone, "pack_revoked", "pack revoked")
		return
	}
	if p.Status != pack.StatusPublished {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url": s.publicOrigin(r) + "/v1/bundles/" + p.ID + "/v/" + strconv.Itoa(p.CurrentVersion),
		"checksum_sha256": p.ChecksumSHA256,
		"version":         p.CurrentVersion,
	})
}

func (s *Server) tombstones(w http.ResponseWriter, r *http.Request) {
	if !s.mustAPIKey(w, r) {
		return
	}
	since := time.Unix(0, 0).UTC()
	if raw := r.URL.Query().Get("since"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			since = parsed
		}
	}
	items, err := s.store.Tombstones(r.Context(), since)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, t := range items {
		out = append(out, map[string]any{
			"pack_id": t.PackID, "revoked_at": t.RevokedAt.UTC().Format(time.RFC3339),
			"reason_code": t.ReasonCode, "checksum_sha256": t.ChecksumSHA256,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (s *Server) createPack(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Name       string   `json:"name"`
		Slug       string   `json:"slug"`
		Author     string   `json:"author"`
		Tags       []string `json:"tags"`
		SourceKind string   `json:"source_kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if !slugPattern.MatchString(body.Slug) || body.Name == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid name or slug")
		return
	}
	kind := body.SourceKind
	if kind != pack.SourceFiles {
		kind = pack.SourceSheet
	}
	author := body.Author
	if author == "" {
		author = u.DisplayName
	}
	p, err := s.store.CreateDraft(r.Context(), u.ID, body.Slug, body.Name, author, body.Tags, kind)
	if err != nil {
		if errors.Is(err, pack.ErrSlugTaken) {
			writeErr(w, http.StatusConflict, "conflict", "slug taken")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": p.ID, "upload_prefix": p.SourcePrefix, "status": p.Status})
}

func (s *Server) updatePack(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	p, err := s.store.GetPack(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanEditPack(p, u.ID, u.IsAdmin()); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", "cannot edit pack")
		return
	}
	var body struct {
		Name string   `json:"name"`
		Slug string   `json:"slug"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Slug = strings.TrimSpace(body.Slug)
	if body.Name == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	slugToUpdate := p.Slug
	if body.Slug != "" && body.Slug != p.Slug {
		if !pack.CanEditSlug(p.Status) {
			writeErr(w, http.StatusConflict, "cannot_edit_slug", "cannot change slug of published or revoked pack")
			return
		}
		if !slugPattern.MatchString(body.Slug) {
			writeErr(w, http.StatusBadRequest, "bad_request", "invalid slug format")
			return
		}
		taken, err := s.store.IsSlugTaken(r.Context(), body.Slug, p.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		if taken {
			writeErr(w, http.StatusConflict, "conflict", "slug taken")
			return
		}
		slugToUpdate = body.Slug
	}

	updated, err := s.store.UpdatePackInfo(r.Context(), p.ID, body.Name, slugToUpdate, body.Tags)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	s.writeDetail(w, r, updated)
}

func (s *Server) updateStickersMetadata(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	p, err := s.store.GetPack(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanEditPack(p, u.ID, u.IsAdmin()); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", "cannot edit pack stickers")
		return
	}

	var body struct {
		Stickers []struct {
			File      string   `json:"file"`
			Filename  string   `json:"filename"`
			EmojiHint string   `json:"emoji_hint"`
			Keywords  []string `json:"keywords"`
		} `json:"stickers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	var items []pack.Sticker
	for _, item := range body.Stickers {
		fn := item.File
		if fn == "" {
			fn = item.Filename
		}
		if fn == "" {
			continue
		}
		items = append(items, pack.Sticker{
			Filename:  fn,
			EmojiHint: item.EmojiHint,
			Keywords:  item.Keywords,
		})
	}

	if len(items) > 0 {
		if err := s.store.UpdateStickersMetadata(r.Context(), p.ID, items); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
	}

	s.writeDetail(w, r, p)
}

func (s *Server) myPacks(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListMine(r.Context(), u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, s.summaryJSON(r, p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) submitPack(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil || p.SubmitterID != u.ID {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if !pack.CanSubmit(p.Status) {
		writeErr(w, http.StatusConflict, "conflict", "cannot submit")
		return
	}
	if err := s.store.EnqueueJob(r.Context(), p); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": pack.StatusProcessing})
}

func (s *Server) confirmPack(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanConfirm(p, u.ID, u.IsAdmin()); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, pack.ErrConflict) {
			status = http.StatusConflict
		}
		writeErr(w, status, "forbidden", err.Error())
		return
	}
	if err := s.store.SetPublished(r.Context(), p.ID, u.ID, "confirm"); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": pack.StatusPublished})
}

func (s *Server) deleteRejectedPack(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanOwnerDeleteRejected(p, u.ID); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, pack.ErrConflict) {
			status = http.StatusConflict
		}
		writeErr(w, status, "forbidden", err.Error())
		return
	}
	if err := s.store.DeleteRejected(r.Context(), p.ID, u.ID); err != nil {
		if errors.Is(err, pack.ErrConflict) {
			writeErr(w, http.StatusConflict, "conflict", "cannot delete pack")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) ownerRevokePack(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanOwnerRevokePublished(p, u.ID); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, pack.ErrConflict) {
			status = http.StatusConflict
		}
		writeErr(w, status, "forbidden", err.Error())
		return
	}
	reason := pack.Reason{
		Code: pack.RevocationCodeAuthorRequest,
		Text: "Author Self-Withdrawal Request: Contributor requested voluntary removal of their pack.",
	}
	if err := s.store.Revoke(r.Context(), p, u.ID, reason); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": pack.StatusRevoked})
}

func (s *Server) adminQueue(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListQueue(r.Context(), u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, s.summaryJSON(r, p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) adminPublished(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.mustAdmin(w, r); !ok {
		return
	}
	items, err := s.store.ListPublishedAdmin(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, s.summaryJSON(r, p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) adminGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.mustAdmin(w, r); !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	s.writeDetail(w, r, p)
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanApprove(p, u.ID, true); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", err.Error())
		return
	}
	if err := s.store.SetPublished(r.Context(), p.ID, u.ID, "approve"); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": pack.StatusPublished})
}

func (s *Server) reject(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	reason, ok := readReason(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := s.store.Reject(r.Context(), p.ID, u.ID, reason); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": pack.StatusRejected})
}

func (s *Server) revoke(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	reason, ok := readReason(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetPack(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "pack not found")
		return
	}
	if err := pack.CanRevoke(p.Status); err != nil && p.Status != pack.StatusRevoked {
		writeErr(w, http.StatusConflict, "conflict", err.Error())
		return
	}
	if err := s.store.Revoke(r.Context(), p, u.ID, reason); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": pack.StatusRevoked})
}

func (s *Server) rejectionAnalytics(w http.ResponseWriter, r *http.Request) {
	_, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	stats, err := s.store.RejectionAnalytics(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"analytics": stats})
}

func readReason(w http.ResponseWriter, r *http.Request) (pack.Reason, bool) {
	var body struct {
		ReasonCode     string `json:"reason_code"`
		RejectionCode  string `json:"rejection_code"`
		RevocationCode string `json:"revocation_code"`
		ReasonText     string `json:"reason_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ReasonText) == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "reason_text required")
		return pack.Reason{}, false
	}
	code := body.RevocationCode
	if code == "" {
		code = body.RejectionCode
	}
	if code == "" {
		code = body.ReasonCode
	}
	if code == "" {
		code = "other"
	}
	return pack.Reason{Code: code, Text: body.ReasonText}, true
}

func (s *Server) writeDetail(w http.ResponseWriter, r *http.Request, p pack.Pack) {
	stickers, err := s.store.Stickers(r.Context(), p.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	list := make([]map[string]any, 0, len(stickers))
	for _, st := range stickers {
		kw := st.Keywords
		if kw == nil {
			kw = []string{}
		}
		list = append(list, map[string]any{
			"file": st.Filename, "emoji_hint": st.EmojiHint, "keywords": kw, "width": st.Width, "height": st.Height,
			"file_url": s.objectURL(r, st.ObjectKey), "thumb_url": s.objectURL(r, st.ThumbKey),
		})
	}
	payload := s.summaryJSON(r, p)
	payload["stickers"] = list
	payload["reject_reason"] = p.RejectReason
	payload["upload_prefix"] = p.SourcePrefix
	if p.ChecksumSHA256 != "" {
		w.Header().Set("ETag", `"sha256:`+p.ChecksumSHA256+`"`)
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) summaryJSON(r *http.Request, p pack.Pack) map[string]any {
	return map[string]any{
		"id": p.ID, "slug": p.Slug, "name": p.Name, "author": p.Author, "tags": p.Tags,
		"sticker_count": p.StickerCount, "version": p.CurrentVersion, "status": p.Status,
		"thumbnail_url": s.objectURL(r, p.ThumbnailKey), "checksum_sha256": p.ChecksumSHA256,
	}
}

func (s *Server) publicOrigin(r *http.Request) string {
	return clientReachableOrigin(s.publicFTP, r)
}

func (s *Server) objectURL(r *http.Request, key string) string {
	if key == "" {
		return ""
	}
	return s.publicOrigin(r) + "/v1/objects/" + strings.TrimPrefix(key, "/")
}

func (s *Server) writeToken(w http.ResponseWriter, u user.User) {
	claims := jwt.MapClaims{
		"sub": u.ID, "role": u.Role, "email": u.Email,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "token failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access_token": signed, "expires_in": 86400, "user": userJSON(u)})
}

func userJSON(u user.User) map[string]any {
	return map[string]any{"id": u.ID, "email": u.Email, "display_name": u.DisplayName, "role": u.Role}
}

func (s *Server) mustUser(w http.ResponseWriter, r *http.Request) (user.User, bool) {
	u, err := s.userFromRequest(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return user.User{}, false
	}
	return u, true
}

func (s *Server) mustAdmin(w http.ResponseWriter, r *http.Request) (user.User, bool) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return user.User{}, false
	}
	if !u.IsAdmin() {
		writeErr(w, http.StatusForbidden, "forbidden", "admin only")
		return user.User{}, false
	}
	return u, true
}

func (s *Server) userFromRequest(r *http.Request) (user.User, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return user.User{}, errors.New("missing bearer")
	}
	token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return user.User{}, errors.New("invalid token")
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	return s.store.GetUser(r.Context(), sub)
}

func (s *Server) mustAPIKey(w http.ResponseWriter, r *http.Request) bool {
	if s.apiKeyOK(r) {
		return true
	}
	writeErr(w, http.StatusUnauthorized, "unauthorized", "api key required")
	return false
}

func (s *Server) apiKeyOK(r *http.Request) bool {
	key := strings.TrimSpace(r.Header.Get("X-Api-Key"))
	if key == "" {
		return false
	}
	ok, err := s.store.APIKeyValid(r.Context(), key)
	return err == nil && ok
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range s.webOrigins {
			if o == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Api-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "86400")
			csp := strings.Join(s.webOrigins, " ")
			w.Header().Set("Content-Security-Policy", "frame-ancestors "+csp)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func (s *Server) listMyAPIKeys(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	keys, err := s.store.ListAPIKeysByUser(r.Context(), u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": keys})
}

func (s *Server) createMyAPIKey(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Name          string `json:"name"`
		ExpiresInDays *int   `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "API Key"
	}
	var expiresAt *time.Time
	if body.ExpiresInDays != nil && *body.ExpiresInDays > 0 {
		exp := time.Now().Add(time.Duration(*body.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}
	rawKey, prefix, hash, err := apikey.GenerateRawKey()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "key generation failed")
		return
	}
	created, err := s.store.CreateAPIKey(r.Context(), &u.ID, name, rawKey, prefix, hash, expiresAt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) rotateMyAPIKey(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	var body struct {
		ExpiresInDays *int `json:"expires_in_days"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var expiresAt *time.Time
	if body.ExpiresInDays != nil && *body.ExpiresInDays > 0 {
		exp := time.Now().Add(time.Duration(*body.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}
	rawKey, prefix, hash, err := apikey.GenerateRawKey()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "key generation failed")
		return
	}
	rotated, err := s.store.RotateAPIKey(r.Context(), id, u.ID, u.IsAdmin(), rawKey, prefix, hash, expiresAt)
	if errors.Is(err, apikey.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not_found", "api key not found")
		return
	}
	if errors.Is(err, apikey.ErrForbidden) {
		writeErr(w, http.StatusForbidden, "forbidden", "cannot rotate api key")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rotated)
}

func (s *Server) deleteMyAPIKey(w http.ResponseWriter, r *http.Request) {
	u, ok := s.mustUser(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	err := s.store.DeleteAPIKey(r.Context(), id, u.ID, u.IsAdmin())
	if errors.Is(err, apikey.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not_found", "api key not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) adminListAPIKeys(w http.ResponseWriter, r *http.Request) {
	_, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	keys, err := s.store.ListAllAPIKeys(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": keys})
}

func (s *Server) adminDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	_, ok := s.mustAdmin(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	err := s.store.DeleteAPIKey(r.Context(), id, "", true)
	if errors.Is(err, apikey.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not_found", "api key not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
