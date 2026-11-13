package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/webp"

	"github.com/eykh/restick/api_svc/internal/animations"
	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/studio"
)

// admin_animations.go serves the animated drafts of one pack: clip upload, the
// draft list, the use toggle, delete, the private preview, the on-white
// composite, the prompt compiler and the publish job.

const (
	// maxMotionTextLength is the accepted motion description length in runes,
	// matching the animations service.
	maxMotionTextLength = 500
	// maxAnimationBody bounds one clip upload: the clip ceiling plus form overhead.
	maxAnimationBody = assets.MaxClipBytes + 64<<10
)

type adminAnimation struct {
	ID                     string            `json:"id"`
	PackID                 string            `json:"pack_id"`
	StickerIndex           int               `json:"sticker_index"`
	MotionText             string            `json:"motion_text"`
	StaticChecksumSHA256   string            `json:"static_checksum_sha256"`
	ClipByteSize           int64             `json:"clip_byte_size"`
	Status                 animations.Status `json:"status"`
	UseAnimated            bool              `json:"use_animated"`
	Stale                  bool              `json:"stale"`
	AnimatedChecksumSHA256 string            `json:"animated_checksum_sha256,omitempty"`
	AnimatedByteSize       int64             `json:"animated_byte_size,omitempty"`
	Frames                 int               `json:"frames,omitempty"`
	FPS                    int               `json:"fps,omitempty"`
	DurationMS             int               `json:"duration_ms,omitempty"`
	Quality                int               `json:"quality,omitempty"`
	Width                  int               `json:"width,omitempty"`
	Height                 int               `json:"height,omitempty"`
	ErrorCode              string            `json:"error_code,omitempty"`
	ErrorMessage           string            `json:"error_message,omitempty"`
	PreviewURL             string            `json:"preview_url"`
}

// adminAnimationSummary is the compact draft state embedded in the pack detail,
// so the admin page renders badges and the publish diff from one call. The
// committed variant is reported separately on the sticker itself.
type adminAnimationSummary struct {
	Status                 animations.Status `json:"status"`
	UseAnimated            bool              `json:"use_animated"`
	Stale                  bool              `json:"stale"`
	PreviewURL             string            `json:"preview_url"`
	AnimatedChecksumSHA256 string            `json:"animated_checksum_sha256,omitempty"`
	AnimatedByteSize       int64             `json:"animated_byte_size,omitempty"`
	Frames                 int               `json:"frames,omitempty"`
	FPS                    int               `json:"fps,omitempty"`
	DurationMS             int               `json:"duration_ms,omitempty"`
	Quality                int               `json:"quality,omitempty"`
	ErrorCode              string            `json:"error_code,omitempty"`
	ErrorMessage           string            `json:"error_message,omitempty"`
}

func (a *admin) animationPreviewURL(packID string, index int) string {
	return fmt.Sprintf("%s/api/v1/admin/packs/%s/stickers/%d/animation/preview", a.baseURL, packID, index)
}

func (a *admin) animation(item animations.Animation) adminAnimation {
	return adminAnimation{
		ID: item.ID, PackID: item.PackID, StickerIndex: item.StickerIndex,
		MotionText: item.MotionText, StaticChecksumSHA256: item.StaticChecksum,
		ClipByteSize: item.ClipByteSize, Status: item.Status, UseAnimated: item.UseAnimated,
		Stale: item.Stale, AnimatedChecksumSHA256: item.AnimatedChecksum,
		AnimatedByteSize: item.AnimatedByteSize, Frames: item.Frames, FPS: item.FPS,
		DurationMS: item.DurationMS, Quality: item.Quality, Width: item.Width, Height: item.Height,
		ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
		PreviewURL: a.animationPreviewURL(item.PackID, item.StickerIndex),
	}
}

func animationSummary(a *admin, item animations.Animation) *adminAnimationSummary {
	return &adminAnimationSummary{
		Status: item.Status, UseAnimated: item.UseAnimated, Stale: item.Stale,
		PreviewURL:             a.animationPreviewURL(item.PackID, item.StickerIndex),
		AnimatedChecksumSHA256: item.AnimatedChecksum, AnimatedByteSize: item.AnimatedByteSize,
		Frames: item.Frames, FPS: item.FPS, DurationMS: item.DurationMS, Quality: item.Quality,
		ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
	}
}

// pathIndex reads the non-negative sticker index of the {index} path segment.
func pathIndex(w http.ResponseWriter, r *http.Request) (int, bool) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || index < 0 {
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "invalid sticker index")
		return 0, false
	}
	return index, true
}

func (a *admin) listAnimations(w http.ResponseWriter, r *http.Request) {
	items, err := a.animations.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAnimationError(w, err)
		return
	}
	out := make([]adminAnimation, 0, len(items))
	for _, item := range items {
		out = append(out, a.animation(item))
	}
	publicJSON(w, http.StatusOK, map[string]any{"items": out})
}

// uploadAnimation streams one multipart clip plus its motion text. The body is
// bounded above the clip ceiling, so a spoofed Content-Length cannot spool more.
func (a *admin) uploadAnimation(w http.ResponseWriter, r *http.Request) {
	packID := r.PathValue("id")
	index, ok := pathIndex(w, r)
	if !ok {
		return
	}
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	boundary := params["boundary"]
	if err != nil || mediaType != "multipart/form-data" || boundary == "" {
		publicError(w, http.StatusUnprocessableEntity, "invalid_content_type", "content type must be multipart/form-data")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAnimationBody)
	reader := multipart.NewReader(r.Body, boundary)
	var clip []byte
	var filename, motionText string
	found := false
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeAnimationReadError(w, err)
			return
		}
		switch part.FormName() {
		case "motion_text":
			value, err := io.ReadAll(io.LimitReader(part, maxUploadField))
			_ = part.Close()
			if err != nil {
				writeAnimationReadError(w, err)
				return
			}
			if motionText == "" {
				motionText = string(value)
			}
		case "clip":
			if found {
				_ = part.Close()
				continue
			}
			filename = part.FileName()
			clip, err = io.ReadAll(io.LimitReader(part, assets.MaxClipBytes+1))
			_ = part.Close()
			if err != nil {
				writeAnimationReadError(w, err)
				return
			}
			if len(clip) > assets.MaxClipBytes {
				publicError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "clip exceeds 20 MiB")
				return
			}
			found = true
		default:
			_ = part.Close()
		}
	}
	if !found {
		publicError(w, http.StatusBadRequest, "bad_request", "missing clip part")
		return
	}
	motionText = strings.TrimSpace(motionText)
	if utf8.RuneCountInString(motionText) > maxMotionTextLength {
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "motion_text exceeds 500 characters")
		return
	}
	item, err := a.animations.Upload(r.Context(), packID, index, bytes.NewReader(clip), filename, motionText)
	if err != nil {
		writeAnimationError(w, err)
		return
	}
	publicJSON(w, http.StatusAccepted, a.animation(item))
}

// writeAnimationReadError maps a failed multipart read; an exhausted body limit
// is always 413, everything else is malformed form data.
func writeAnimationReadError(w http.ResponseWriter, err error) {
	if tooLarge(err) {
		publicError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "upload exceeds the clip ceiling plus form overhead")
		return
	}
	publicError(w, http.StatusBadRequest, "bad_request", "invalid multipart body")
}

func (a *admin) setAnimationUse(w http.ResponseWriter, r *http.Request) {
	index, ok := pathIndex(w, r)
	if !ok {
		return
	}
	var body struct {
		UseAnimated *bool `json:"use_animated"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.UseAnimated == nil {
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "use_animated is required")
		return
	}
	item, err := a.animations.SetUse(r.Context(), r.PathValue("id"), index, *body.UseAnimated)
	if err != nil {
		writeAnimationError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, a.animation(item))
}

func (a *admin) deleteAnimation(w http.ResponseWriter, r *http.Request) {
	index, ok := pathIndex(w, r)
	if !ok {
		return
	}
	if err := a.animations.Delete(r.Context(), r.PathValue("id"), index); err != nil {
		writeAnimationError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// animationDraft finds one draft through the list query, which is also the only
// place stale is computed against the current version.
func (a *admin) animationDraft(ctx context.Context, packID string, index int) (animations.Animation, error) {
	items, err := a.animations.List(ctx, packID)
	if err != nil {
		return animations.Animation{}, err
	}
	for _, item := range items {
		if item.StickerIndex == index {
			return item, nil
		}
	}
	return animations.Animation{}, animations.ErrNotFound
}

// getAnimationPreview serves the private draft WebP through the safe opener.
// Only a ready draft has a preview: a queued or failed clip answers 409.
func (a *admin) getAnimationPreview(w http.ResponseWriter, r *http.Request) {
	index, ok := pathIndex(w, r)
	if !ok {
		return
	}
	item, err := a.animationDraft(r.Context(), r.PathValue("id"), index)
	if err != nil {
		writeAnimationError(w, err)
		return
	}
	if item.Status != animations.StatusReady || item.AnimatedPath == "" {
		publicError(w, http.StatusConflict, "not_ready", "animation is not ready")
		return
	}
	file, err := a.assets.OpenAnimated(item.AnimatedPath)
	if err != nil {
		if errors.Is(err, assets.ErrNotFound) {
			publicError(w, http.StatusNotFound, "not_found", "animation preview not found")
			return
		}
		publicError(w, http.StatusInternalServerError, "internal", "could not open animation preview")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not open animation preview")
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", info.ModTime(), file)
}

// getOnWhite composites the committed static sticker over opaque white as PNG:
// image models reject transparent inputs, so the operator gets a white-backed
// copy without changing the published asset.
func (a *admin) getOnWhite(w http.ResponseWriter, r *http.Request) {
	index, ok := pathIndex(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	pack, err := a.packs.Get(ctx, r.PathValue("id"))
	if err != nil {
		writePackError(w, err)
		return
	}
	if pack.CurrentVersion < 1 {
		publicError(w, http.StatusNotFound, "not_found", "pack has no published version")
		return
	}
	version, err := a.packs.GetVersion(ctx, pack.ID, pack.CurrentVersion)
	if err != nil {
		writePackError(w, err)
		return
	}
	sticker, ok := findSticker(version.Stickers, index)
	if !ok {
		publicError(w, http.StatusNotFound, "not_found", "sticker not found")
		return
	}
	file, err := a.assets.OpenCommitted(sticker.ObjectPath)
	if err != nil {
		if errors.Is(err, assets.ErrNotFound) {
			publicError(w, http.StatusNotFound, "not_found", "sticker file not found")
			return
		}
		publicError(w, http.StatusInternalServerError, "internal", "could not open sticker")
		return
	}
	defer file.Close()
	source, err := webp.Decode(file)
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not decode sticker")
		return
	}
	canvas := image.NewRGBA(image.Rect(0, 0, assets.StickerDimension, assets.StickerDimension))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), source, source.Bounds().Min, draw.Over)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not encode sticker")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded.Bytes())
}

func findSticker(stickers []packs.Sticker, index int) (packs.Sticker, bool) {
	for _, sticker := range stickers {
		if sticker.SortOrder == index {
			return sticker, true
		}
	}
	return packs.Sticker{}, false
}

// compileAnimation validates the sticker against the current version and
// returns the English prompt plus its emoji hint; nothing is written.
func (a *admin) compileAnimation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PackID       string `json:"pack_id"`
		StickerIndex int    `json:"sticker_index"`
		MotionText   string `json:"motion_text"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	motionText := strings.TrimSpace(body.MotionText)
	if body.StickerIndex < 0 || utf8.RuneCountInString(motionText) > maxMotionTextLength {
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "invalid sticker index or motion_text")
		return
	}
	ctx := r.Context()
	pack, err := a.packs.Get(ctx, body.PackID)
	if err != nil {
		writePackError(w, err)
		return
	}
	if pack.CurrentVersion < 1 {
		publicError(w, http.StatusNotFound, "not_found", "pack has no published version")
		return
	}
	version, err := a.packs.GetVersion(ctx, pack.ID, pack.CurrentVersion)
	if err != nil {
		writePackError(w, err)
		return
	}
	sticker, ok := findSticker(version.Stickers, body.StickerIndex)
	if !ok {
		publicError(w, http.StatusNotFound, "not_found", "sticker not found")
		return
	}
	prompt, err := a.studio.CompileAnimation(ctx, pack.DesignID, motionText)
	if err != nil {
		writeAnimationError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, map[string]string{"prompt": prompt, "emoji_hint": sticker.EmojiHint})
}

// publishAnimations queues the Go-only republish job; the contract check and the
// animated-draft conflicts are mapped by writeAnimationError.
func (a *admin) publishAnimations(w http.ResponseWriter, r *http.Request) {
	job, err := a.jobs.EnqueuePublish(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAnimationError(w, err)
		return
	}
	publicJSON(w, http.StatusAccepted, jobJSON(job))
}

// writeAnimationError maps animation draft, publish job and store failures.
func writeAnimationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, animations.ErrNotFound), errors.Is(err, jobs.ErrNotFound),
		errors.Is(err, packs.ErrNotFound), errors.Is(err, packs.ErrVersionNotFound),
		errors.Is(err, studio.ErrNotFound):
		publicError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, jobs.ErrAnimationBusy):
		publicError(w, http.StatusConflict, "animation_busy", "an animation clip is still processing")
	case errors.Is(err, jobs.ErrNoChanges):
		publicError(w, http.StatusConflict, "no_changes", "no animation changes to publish")
	case errors.Is(err, animations.ErrJobActive), errors.Is(err, jobs.ErrJobActive),
		errors.Is(err, packs.ErrJobActive):
		publicError(w, http.StatusConflict, "job_active", "pack has a queued or running job")
	case errors.Is(err, animations.ErrNotReady):
		publicError(w, http.StatusConflict, "not_ready", "animation is not ready")
	case errors.Is(err, packs.ErrRevoked), errors.Is(err, jobs.ErrPackRevoked):
		publicError(w, http.StatusGone, "pack_revoked", "pack is revoked")
	case errors.Is(err, animations.ErrInvalid), errors.Is(err, jobs.ErrInvalid),
		errors.Is(err, packs.ErrInvalid), errors.Is(err, studio.ErrInvalid):
		publicError(w, http.StatusUnprocessableEntity, "unprocessable", "invalid request")
	default:
		publicError(w, http.StatusInternalServerError, "internal", "could not complete request")
	}
}
