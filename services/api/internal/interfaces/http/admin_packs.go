package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/animations"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
)

// admin_packs.go serves pack CRUD, source upload, processing, revoke and job
// inspection. Mutations return the affected domain object.

type adminPackSheet struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	Filename       string `json:"filename"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ByteSize       int64  `json:"byte_size"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	ImageURL       string `json:"image_url"`
}

type adminPackSticker struct {
	SortOrder      int      `json:"sort_order"`
	Filename       string   `json:"filename"`
	ChecksumSHA256 string   `json:"checksum_sha256"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	EmojiHint      string   `json:"emoji_hint"`
	Keywords       []string `json:"keywords"`
	ObjectURL      string   `json:"object_url"`
	ThumbURL       string   `json:"thumb_url"`
	// Animated is the committed variant of this version, nil for a still
	// sticker. It is what clients actually download, so the admin page can
	// compare it with the draft Animation below and show unpublished changes.
	Animated *adminPackAnimated `json:"animated,omitempty"`
	// Animation is the mutable draft for this sticker index, if any.
	Animation *adminAnimationSummary `json:"animation,omitempty"`
}

type adminPackAnimated struct {
	ObjectURL      string `json:"object_url"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	ByteSize       int64  `json:"byte_size"`
	Frames         int    `json:"frames"`
	DurationMS     int    `json:"duration_ms"`
}

type adminPack struct {
	ID                  string          `json:"id"`
	Slug                string          `json:"slug"`
	Name                string          `json:"name"`
	Author              string          `json:"author"`
	Description         string          `json:"description"`
	Tags                []string        `json:"tags"`
	DesignID            string          `json:"design_id,omitempty"`
	Status              packs.Status    `json:"status"`
	CurrentVersion      int             `json:"current_version"`
	StickerCount        int             `json:"sticker_count"`
	ChecksumSHA256      string          `json:"checksum_sha256"`
	ThumbnailURL        string          `json:"thumbnail_url"`
	SheetID             string          `json:"sheet_id"`
	Sheet               *adminPackSheet `json:"sheet"`
	ThemeColor          string          `json:"theme_color"`
	ThemeColorSuggested string          `json:"theme_color_suggested"`
	ThemeColorEffective string          `json:"theme_color_effective"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

type adminPackDesign struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Rows           int    `json:"rows"`
	Columns        int    `json:"columns"`
	OutlineWidth   int    `json:"outline_width"`
	OutlineColor   string `json:"outline_color"`
	CharacterID    string `json:"character_id,omitempty"`
	CompiledPrompt string `json:"compiled_prompt"`
}

type adminPackDetail struct {
	adminPack
	Design    *adminPackDesign   `json:"design,omitempty"`
	Stickers  []adminPackSticker `json:"stickers"`
	BundleURL string             `json:"bundle_url"`
	LatestJob *adminJob          `json:"latest_job,omitempty"`
}

func (a *admin) pack(p packs.Pack) adminPack {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	out := adminPack{
		ID: p.ID, Slug: p.Slug, Name: p.Name, Author: p.Author, Description: p.Description,
		Tags: tags, DesignID: p.DesignID, Status: p.Status, CurrentVersion: p.CurrentVersion,
		StickerCount: p.StickerCount, ChecksumSHA256: p.ChecksumSHA256,
		ThumbnailURL: a.assetURL(p.ThumbnailPath),
		CreatedAt:    formatSeconds(p.CreatedAt), UpdatedAt: formatSeconds(p.UpdatedAt),
	}
	out.SheetID = p.SheetID
	out.ThemeColor, out.ThemeColorSuggested, out.ThemeColorEffective = p.ThemeColor, p.ThemeColorSuggested, p.EffectiveThemeColor()
	if p.Sheet != nil {
		out.Sheet = &adminPackSheet{
			ID: p.Sheet.ID, Provider: p.Sheet.Provider, Filename: p.Sheet.Filename,
			Width: p.Sheet.Width, Height: p.Sheet.Height, ByteSize: p.Sheet.ByteSize,
			ChecksumSHA256: p.Sheet.ChecksumSHA256,
			ImageURL:       a.sheetImageURL(p.DesignID, p.Sheet.ID),
		}
	}
	return out
}

func (a *admin) assetURL(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	return a.baseURL + "/v1/objects/" + strings.TrimPrefix(relativePath, "/")
}

func formatSeconds(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func optionalSeconds(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	value := formatSeconds(t)
	return &value
}

func (a *admin) listPacks(w http.ResponseWriter, r *http.Request) {
	page, err := a.packs.ListAll(r.Context(), packs.AdminFilter{
		Query:    r.URL.Query().Get("q"),
		Status:   r.URL.Query().Get("status"),
		Tag:      r.URL.Query().Get("tag"),
		Page:     queryInt(r, "page"),
		PageSize: queryInt(r, "page_size"),
	})
	if err != nil {
		if errors.Is(err, packs.ErrInvalid) {
			publicError(w, http.StatusUnprocessableEntity, "unprocessable", "unknown status filter")
			return
		}
		publicError(w, http.StatusInternalServerError, "internal", "could not list packs")
		return
	}
	items := make([]adminPack, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, a.pack(item))
	}
	publicJSON(w, http.StatusOK, map[string]any{
		"items": items, "page": page.Page, "page_size": page.PageSize, "total": page.Total,
	})
}

func (a *admin) createPack(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug        string   `json:"slug"`
		Name        string   `json:"name"`
		Author      string   `json:"author"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		DesignID    string   `json:"design_id"`
		SheetID     string   `json:"sheet_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	pack, err := a.packs.Create(r.Context(), packs.PackDraft{
		Slug: body.Slug, Name: body.Name, Author: body.Author,
		Description: body.Description, Tags: body.Tags, DesignID: body.DesignID, SheetID: body.SheetID,
	})
	if err != nil {
		writePackError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, a.pack(pack))
}

// getPack is the admin detail: current version stickers plus the bundle URL.
// Draft and revoked packs stay visible here only.
func (a *admin) getPack(w http.ResponseWriter, r *http.Request) {
	pack, err := a.packs.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writePackError(w, err)
		return
	}
	detail := adminPackDetail{adminPack: a.pack(pack), Stickers: []adminPackSticker{}}
	if pack.DesignID != "" {
		design, err := a.studio.GetDesign(r.Context(), pack.DesignID)
		if err != nil {
			writeStudioError(w, err)
			return
		}
		detail.Design = &adminPackDesign{
			ID: design.ID, Name: design.Name, Rows: design.Rows, Columns: design.Columns,
			OutlineWidth: design.OutlineWidth, OutlineColor: design.OutlineColor,
			CharacterID: design.CharacterID, CompiledPrompt: design.Prompt,
		}
	}
	latest, err := a.jobs.LatestForPack(r.Context(), pack.ID)
	if err != nil {
		writePackError(w, err)
		return
	}
	if latest != nil {
		job := jobJSON(*latest)
		detail.LatestJob = &job
	}
	if pack.CurrentVersion > 0 {
		version, err := a.packs.GetVersion(r.Context(), pack.ID, pack.CurrentVersion)
		if err != nil {
			writePackError(w, err)
			return
		}
		drafts := map[int]animations.Animation{}
		if a.animations != nil {
			items, err := a.animations.List(r.Context(), pack.ID)
			if err != nil {
				writeAnimationError(w, err)
				return
			}
			for _, item := range items {
				drafts[item.StickerIndex] = item
			}
		}
		for _, sticker := range version.Stickers {
			keywords := sticker.Keywords
			if keywords == nil {
				keywords = []string{}
			}
			entry := adminPackSticker{
				SortOrder: sticker.SortOrder, Filename: sticker.Filename,
				ChecksumSHA256: sticker.ChecksumSHA256, Width: sticker.Width, Height: sticker.Height,
				EmojiHint: sticker.EmojiHint, Keywords: keywords,
				ObjectURL: a.assetURL(sticker.ObjectPath), ThumbURL: a.assetURL(sticker.ThumbPath),
			}
			if sticker.AnimatedPath != "" {
				entry.Animated = &adminPackAnimated{
					ObjectURL: a.assetURL(sticker.AnimatedPath), ChecksumSHA256: sticker.AnimatedChecksumSHA256,
					ByteSize: sticker.AnimatedByteSize, Frames: sticker.AnimatedFrames,
					DurationMS: sticker.AnimatedDurationMS,
				}
			}
			if item, ok := drafts[sticker.SortOrder]; ok {
				entry.Animation = animationSummary(a, item)
			}
			detail.Stickers = append(detail.Stickers, entry)
		}
		detail.BundleURL = a.baseURL + "/v1/bundles/" + pack.ID + "/v/" + strconv.Itoa(pack.CurrentVersion)
	}
	publicJSON(w, http.StatusOK, detail)
}

func (a *admin) updatePack(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string   `json:"name"`
		Author      string   `json:"author"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		DesignID    string   `json:"design_id"`
		SheetID     string   `json:"sheet_id"`
		ThemeColor  *string  `json:"theme_color"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	pack, err := a.packs.UpdateCatalog(r.Context(), r.PathValue("id"), packs.Metadata{
		Name: body.Name, Author: body.Author, Description: body.Description,
		Tags: body.Tags, DesignID: body.DesignID, SheetID: body.SheetID, ThemeColor: body.ThemeColor,
	})
	if err != nil {
		writePackError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, a.pack(pack))
}

func (a *admin) deletePack(w http.ResponseWriter, r *http.Request) {
	if err := a.packs.DeleteDraft(r.Context(), r.PathValue("id")); err != nil {
		writePackError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *admin) processPack(w http.ResponseWriter, r *http.Request) {
	job, err := a.jobs.Enqueue(r.Context(), r.PathValue("id"), "", "")
	if err != nil {
		writePackError(w, err)
		return
	}
	publicJSON(w, http.StatusAccepted, jobJSON(job))
}

// revokePack is idempotent: a second call keeps the single tombstone.
func (a *admin) revokePack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.jobs.Revoke(r.Context(), id); err != nil {
		writePackError(w, err)
		return
	}
	pack, err := a.packs.Get(r.Context(), id)
	if err != nil {
		writePackError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, a.pack(pack))
}

func (a *admin) getJob(w http.ResponseWriter, r *http.Request) {
	job, err := a.jobs.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writePackError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, jobJSON(job))
}

type adminJob struct {
	ID           string          `json:"id"`
	PackID       string          `json:"pack_id"`
	Version      int             `json:"version"`
	Status       jobs.Status     `json:"status"`
	Request      jobs.JobRequest `json:"request"`
	Result       json.RawMessage `json:"result"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	Diagnostics  string          `json:"diagnostics,omitempty"`
	CreatedAt    string          `json:"created_at"`
	StartedAt    *string         `json:"started_at"`
	FinishedAt   *string         `json:"finished_at"`
}

func jobJSON(job jobs.Job) adminJob {
	return adminJob{
		ID: job.ID, PackID: job.PackID, Version: job.Version, Status: job.Status,
		Request: job.Request, Result: job.ResultJSON,
		ErrorCode: job.ErrorCode, ErrorMessage: job.ErrorMessage, Diagnostics: job.Diagnostics,
		CreatedAt: formatSeconds(job.CreatedAt),
		StartedAt: optionalSeconds(job.StartedAt), FinishedAt: optionalSeconds(job.FinishedAt),
	}
}
