package assets

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/image/riff"
	"golang.org/x/image/webp"
)

const (
	// PackSchema is the immutable manifest schema version written into pack.json.
	PackSchema = "restick.pack.v1"
	// StickerDimension is the fixed square edge of every committed sticker.
	StickerDimension = 512
	// ThumbnailDimension is the fixed square edge of every sticker thumbnail.
	ThumbnailDimension = 128
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// PackManifest is the snapshot used to build one committed version. It is
// stored verbatim inside pack.json and never edited by catalog updates.
type PackManifest struct {
	PackID      string
	Slug        string
	Name        string
	Author      string
	Description string
	Tags        []string
	Version     int
	Stickers    []ManifestSticker
}

// ManifestSticker describes one processed cell. File and ThumbFile are flat
// names inside work/<job>/output; committed names are derived from their order.
type ManifestSticker struct {
	File      string
	ThumbFile string
	Width     int
	Height    int
	EmojiHint string
	Keywords  []string
	Animated  *ManifestAnimated
}

// ManifestAnimated is the optional animated variant of a sticker. File is a
// flat name inside work/<job>/output/animated/ and is committed as
// animated/<File>; Frames and DurationMS are reported by media_factory.
type ManifestAnimated struct {
	File       string
	Frames     int
	DurationMS int
	Width      int
	Height     int
}

// StagedVersion is a fully written version that is not public yet: pack.json,
// stickers, thumbnails and the ZIP live under work/<job>/stage-*.
type StagedVersion struct {
	JobID          string
	PackID         string
	Version        int
	Manifest       PackManifest
	ChecksumSHA256 string
	BundleByteSize int64
	StickerCount   int

	stageDir     string
	stickers     []CommittedSticker
	manifestJSON []byte
}

// CommittedSticker is one immutable sticker row with data-root-relative paths.
type CommittedSticker struct {
	SortOrder      int
	Filename       string
	ObjectPath     string
	ThumbPath      string
	ChecksumSHA256 string
	Width          int
	Height         int
	EmojiHint      string
	Keywords       []string

	// Animated* stay zero for a sticker without an animated variant.
	AnimatedPath           string
	AnimatedChecksumSHA256 string
	AnimatedByteSize       int64
	AnimatedFrames         int
	AnimatedDurationMS     int
}

// CommittedFiles lists the files renamed into their final public locations.
// Every path is relative to the data root and belongs in the database.
type CommittedFiles struct {
	JobID          string
	PackID         string
	Version        int
	ChecksumSHA256 string
	StickerCount   int
	ManifestJSON   []byte
	BundlePath     string
	BundleByteSize int64
	ThumbnailPath  string
	OutputDir      string
	Stickers       []CommittedSticker
}

// PackJSON renders the immutable pack.json snapshot with its aggregate
// checksum (SHA256 over sticker bytes in manifest order, each static file
// followed by its animated file when present; never the ZIP).
func (m PackManifest) PackJSON(checksum string) ([]byte, error) {
	doc := packDocument{
		Schema:         PackSchema,
		ID:             m.PackID,
		Slug:           m.Slug,
		Name:           m.Name,
		Author:         m.Author,
		Description:    m.Description,
		Tags:           m.Tags,
		IPPolicy:       "original_only",
		Version:        m.Version,
		ChecksumSHA256: checksum,
		Stickers:       make([]packDocumentSticker, len(m.Stickers)),
	}
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	for i, sticker := range m.Stickers {
		doc.Stickers[i] = packDocumentSticker{
			File:      sticker.File,
			Thumb:     path.Join("thumbs", sticker.ThumbFile),
			EmojiHint: sticker.EmojiHint,
			Width:     sticker.Width,
			Height:    sticker.Height,
		}
		if a := sticker.Animated; a != nil {
			doc.Stickers[i].Animated = &packDocumentAnimated{
				File:       path.Join("animated", a.File),
				Frames:     a.Frames,
				DurationMS: a.DurationMS,
				Width:      a.Width,
				Height:     a.Height,
			}
		}
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("assets: encode pack.json: %w", err)
	}
	return append(body, '\n'), nil
}

type packDocument struct {
	Schema         string                `json:"schema"`
	ID             string                `json:"id"`
	Slug           string                `json:"slug"`
	Name           string                `json:"name"`
	Author         string                `json:"author"`
	Description    string                `json:"description"`
	Tags           []string              `json:"tags"`
	IPPolicy       string                `json:"ip_policy"`
	Version        int                   `json:"version"`
	ChecksumSHA256 string                `json:"checksum_sha256"`
	Stickers       []packDocumentSticker `json:"stickers"`
}

type packDocumentSticker struct {
	File      string `json:"file"`
	Thumb     string `json:"thumb"`
	EmojiHint string `json:"emoji_hint"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	// omitempty keeps a static pack.json byte-identical to the pre-animation format.
	Animated *packDocumentAnimated `json:"animated,omitempty"`
}

type packDocumentAnimated struct {
	File       string `json:"file"`
	Frames     int    `json:"frames"`
	DurationMS int    `json:"duration_ms"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// StageBundle copies validated job output into a staged version directory and
// streams the ZIP to disk. Nothing becomes visible until CommitFiles renames it
// and the caller commits the database rows.
func (s *Store) StageBundle(ctx context.Context, jobID string, manifest PackManifest) (StagedVersion, error) {
	if !jobIDPattern.MatchString(jobID) {
		return StagedVersion{}, ErrInvalidJobID
	}
	if err := manifest.validate(); err != nil {
		return StagedVersion{}, err
	}
	if err := ctx.Err(); err != nil {
		return StagedVersion{}, err
	}
	workDir, err := s.WorkDir(jobID)
	if err != nil {
		return StagedVersion{}, err
	}
	outputDir, err := s.path("work", jobID, "output")
	if err != nil {
		return StagedVersion{}, err
	}
	if err := s.verifyDir(outputDir); err != nil {
		return StagedVersion{}, err
	}
	stageDir, err := os.MkdirTemp(workDir, "stage-")
	if err != nil {
		return StagedVersion{}, fmt.Errorf("assets: create stage dir: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stageDir)
		}
	}()
	packDir := filepath.Join(stageDir, "pack")
	if err := os.MkdirAll(filepath.Join(packDir, "thumbs"), 0o755); err != nil {
		return StagedVersion{}, fmt.Errorf("assets: create staged pack: %w", err)
	}
	syncDirs := []string{packDir, filepath.Join(packDir, "thumbs"), stageDir}
	animatedOutput := filepath.Join(outputDir, "animated")
	if manifest.hasAnimated() {
		if err := s.verifyDir(animatedOutput); err != nil {
			return StagedVersion{}, memberError("animated", err)
		}
		if err := os.Mkdir(filepath.Join(packDir, "animated"), 0o755); err != nil {
			return StagedVersion{}, fmt.Errorf("assets: create staged pack: %w", err)
		}
		syncDirs = append(syncDirs, filepath.Join(packDir, "animated"))
	}

	aggregate := sha256.New()
	stickers := make([]CommittedSticker, len(manifest.Stickers))
	budget := newCopyBudget(MaxBundleBytes)
	for i, sticker := range manifest.Stickers {
		if err := ctx.Err(); err != nil {
			return StagedVersion{}, err
		}
		objectPath := filepath.Join(outputDir, sticker.File)
		stagedObject := filepath.Join(packDir, sticker.File)
		sum, _, err := copyFile(stagedObject, objectPath, aggregate, budget)
		if err != nil {
			return StagedVersion{}, memberError(sticker.File, err)
		}
		if err := verifyWebP(stagedObject, StickerDimension, StickerDimension); err != nil {
			return StagedVersion{}, memberError(sticker.File, err)
		}
		thumbPath := filepath.Join(outputDir, sticker.ThumbFile)
		stagedThumb := filepath.Join(packDir, "thumbs", sticker.ThumbFile)
		if _, _, err := copyFile(stagedThumb, thumbPath, nil, budget); err != nil {
			return StagedVersion{}, memberError(sticker.ThumbFile, err)
		}
		if err := verifyWebP(stagedThumb, ThumbnailDimension, ThumbnailDimension); err != nil {
			return StagedVersion{}, memberError(sticker.ThumbFile, err)
		}
		if i == 0 {
			if _, _, err := copyFile(filepath.Join(packDir, "thumb.webp"), stagedThumb, nil, budget); err != nil {
				return StagedVersion{}, memberError("thumb.webp", err)
			}
		}
		stickers[i] = CommittedSticker{
			SortOrder:      i,
			Filename:       sticker.File,
			ChecksumSHA256: sum,
			Width:          sticker.Width,
			Height:         sticker.Height,
			EmojiHint:      sticker.EmojiHint,
			Keywords:       append([]string(nil), sticker.Keywords...),
		}
		if a := sticker.Animated; a != nil {
			member := path.Join("animated", a.File)
			stagedAnimated := filepath.Join(packDir, "animated", a.File)
			// Appended right after this sticker's static bytes (spec "pack.json và checksum").
			animatedSum, n, err := copyFile(stagedAnimated, filepath.Join(animatedOutput, a.File), aggregate, budget)
			if err != nil {
				return StagedVersion{}, memberError(member, err)
			}
			if err := verifyAnimatedWebP(stagedAnimated, StickerDimension, StickerDimension); err != nil {
				return StagedVersion{}, memberError(member, err)
			}
			stickers[i].AnimatedChecksumSHA256 = animatedSum
			stickers[i].AnimatedByteSize = n
			stickers[i].AnimatedFrames = a.Frames
			stickers[i].AnimatedDurationMS = a.DurationMS
		}
	}
	checksum := hex.EncodeToString(aggregate.Sum(nil))
	manifestJSON, err := manifest.PackJSON(checksum)
	if err != nil {
		return StagedVersion{}, err
	}
	if err := writeFileSync(filepath.Join(packDir, "pack.json"), manifestJSON); err != nil {
		return StagedVersion{}, err
	}

	bundleStage := filepath.Join(stageDir, "bundle.zip")
	bundleSize, err := writeBundle(packDir, bundleStage, manifest)
	if err != nil {
		return StagedVersion{}, err
	}
	for _, dir := range syncDirs {
		if err := syncDir(dir); err != nil {
			return StagedVersion{}, err
		}
	}
	committed = true
	return StagedVersion{
		JobID:          jobID,
		PackID:         manifest.PackID,
		Version:        manifest.Version,
		Manifest:       manifest,
		ChecksumSHA256: checksum,
		BundleByteSize: bundleSize,
		StickerCount:   len(stickers),
		stageDir:       stageDir,
		stickers:       stickers,
		manifestJSON:   manifestJSON,
	}, nil
}

// CommitFiles renames a staged version into packs/<id>/v<n> and bundles, then
// fsyncs the parent directories. It fails instead of overwriting a version.
func (s *Store) CommitFiles(ctx context.Context, staged StagedVersion) (CommittedFiles, error) {
	if err := ctx.Err(); err != nil {
		return CommittedFiles{}, err
	}
	if !idPattern.MatchString(staged.PackID) || staged.Version < 1 || staged.stageDir == "" {
		return CommittedFiles{}, ErrInvalidManifest
	}
	if !s.contains(staged.stageDir) {
		return CommittedFiles{}, ErrUnsafePath
	}
	packParent, err := s.path("packs", staged.PackID)
	if err != nil {
		return CommittedFiles{}, err
	}
	packDir := filepath.Join(packParent, fmt.Sprintf("v%d", staged.Version))
	bundleParent, err := s.path("bundles", staged.PackID)
	if err != nil {
		return CommittedFiles{}, err
	}
	bundlePath := filepath.Join(bundleParent, fmt.Sprintf("v%d.zip", staged.Version))
	if _, err := os.Lstat(packDir); err == nil {
		return CommittedFiles{}, ErrAlreadyCommitted
	}
	if _, err := os.Lstat(bundlePath); err == nil {
		return CommittedFiles{}, ErrAlreadyCommitted
	}
	if err := os.MkdirAll(packParent, 0o755); err != nil {
		return CommittedFiles{}, fmt.Errorf("assets: create packs dir: %w", err)
	}
	if err := os.MkdirAll(bundleParent, 0o755); err != nil {
		return CommittedFiles{}, fmt.Errorf("assets: create bundles dir: %w", err)
	}
	if err := os.Rename(filepath.Join(staged.stageDir, "pack"), packDir); err != nil {
		return CommittedFiles{}, fmt.Errorf("assets: commit pack files: %w", err)
	}
	if err := os.Rename(filepath.Join(staged.stageDir, "bundle.zip"), bundlePath); err != nil {
		// The pack directory stays an unpublished orphan; the version is not
		// referenced by the database.
		return CommittedFiles{}, fmt.Errorf("assets: commit bundle: %w", err)
	}
	for _, dir := range []string{packParent, bundleParent, filepath.Join(s.dataDir, "packs"), filepath.Join(s.dataDir, "bundles")} {
		if err := syncDir(dir); err != nil {
			return CommittedFiles{}, err
		}
	}
	_ = os.RemoveAll(staged.stageDir)

	outputDir := path.Join("packs", staged.PackID, fmt.Sprintf("v%d", staged.Version))
	committedStickers := make([]CommittedSticker, len(staged.stickers))
	for i, sticker := range staged.stickers {
		sticker.ObjectPath = path.Join(outputDir, sticker.Filename)
		sticker.ThumbPath = path.Join(outputDir, "thumbs", staged.Manifest.Stickers[i].ThumbFile)
		if a := staged.Manifest.Stickers[i].Animated; a != nil {
			sticker.AnimatedPath = path.Join(outputDir, "animated", a.File)
		}
		committedStickers[i] = sticker
	}
	return CommittedFiles{
		JobID:          staged.JobID,
		PackID:         staged.PackID,
		Version:        staged.Version,
		ChecksumSHA256: staged.ChecksumSHA256,
		StickerCount:   staged.StickerCount,
		ManifestJSON:   staged.manifestJSON,
		BundlePath:     path.Join("bundles", staged.PackID, fmt.Sprintf("v%d.zip", staged.Version)),
		BundleByteSize: staged.BundleByteSize,
		ThumbnailPath:  path.Join(outputDir, "thumb.webp"),
		OutputDir:      outputDir,
		Stickers:       committedStickers,
	}, nil
}

func (m PackManifest) validate() error {
	switch {
	case !idPattern.MatchString(m.PackID):
		return fmt.Errorf("%w: invalid pack id", ErrInvalidManifest)
	case !slugPattern.MatchString(m.Slug):
		return fmt.Errorf("%w: invalid slug", ErrInvalidManifest)
	case strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Author) == "":
		return fmt.Errorf("%w: name and author are required", ErrInvalidManifest)
	case m.Version < 1:
		return fmt.Errorf("%w: invalid version", ErrInvalidManifest)
	case len(m.Stickers) == 0 || len(m.Stickers) > MaxStickers:
		return fmt.Errorf("%w: sticker count %d", ErrInvalidManifest, len(m.Stickers))
	}
	for i, sticker := range m.Stickers {
		if sticker.File != fmt.Sprintf("%03d.webp", i) {
			return fmt.Errorf("%w: member %q is not the fixed name for index %d", ErrInvalidManifest, sticker.File, i)
		}
		if sticker.ThumbFile != fmt.Sprintf("thumb_%03d.webp", i) {
			return fmt.Errorf("%w: member %q is not the fixed thumb name for index %d", ErrInvalidManifest, sticker.ThumbFile, i)
		}
		if sticker.Width != StickerDimension || sticker.Height != StickerDimension {
			return fmt.Errorf("%w: %s must be declared %dx%d, got %dx%d",
				ErrInvalidManifest, sticker.File,
				StickerDimension, StickerDimension, sticker.Width, sticker.Height)
		}
		for _, keyword := range sticker.Keywords {
			if strings.TrimSpace(keyword) == "" {
				return fmt.Errorf("%w: empty keyword for %s", ErrInvalidManifest, sticker.File)
			}
		}
		if a := sticker.Animated; a != nil {
			// The fixed name rules out traversal and collisions inside animated/.
			if a.File != sticker.File {
				return fmt.Errorf("%w: animated member %q is not the fixed name for index %d", ErrInvalidManifest, a.File, i)
			}
			if a.Width != StickerDimension || a.Height != StickerDimension {
				return fmt.Errorf("%w: animated %s must be declared %dx%d, got %dx%d",
					ErrInvalidManifest, a.File, StickerDimension, StickerDimension, a.Width, a.Height)
			}
			if a.Frames < 2 || a.DurationMS < 1 {
				return fmt.Errorf("%w: animated %s needs at least 2 frames and a duration", ErrInvalidManifest, a.File)
			}
		}
	}
	return nil
}

func (m PackManifest) hasAnimated() bool {
	for _, sticker := range m.Stickers {
		if sticker.Animated != nil {
			return true
		}
	}
	return false
}

// verifyWebP checks the staged bytes: the decoded edge must be exactly the
// required output size, so a self-consistent but wrong-size processor result is
// rejected before anything becomes public.
func verifyWebP(p string, width, height int) error {
	f, err := openNoFollow(p)
	if err != nil {
		return err
	}
	defer f.Close()
	cfg, err := webp.DecodeConfig(f)
	if err != nil {
		return ErrInvalidImage
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxImageDimension || cfg.Height > MaxImageDimension {
		return ErrImageDimensions
	}
	if width > 0 && cfg.Width != width {
		return fmt.Errorf("%w: width %d does not match manifest %d", ErrInvalidManifest, cfg.Width, width)
	}
	if height > 0 && cfg.Height != height {
		return fmt.Errorf("%w: height %d does not match manifest %d", ErrInvalidManifest, cfg.Height, height)
	}
	return nil
}

// verifyAnimatedWebP checks a staged animated WebP without decoding pixels:
// x/image/webp cannot decode animations, so the canvas comes from DecodeConfig
// (the VP8X chunk) and the frames are counted as ANMF chunks.
func verifyAnimatedWebP(p string, width, height int) error {
	if err := verifyWebP(p, width, height); err != nil {
		return err
	}
	f, err := openNoFollow(p)
	if err != nil {
		return err
	}
	defer f.Close()
	formType, chunks, err := riff.NewReader(f)
	if err != nil || formType != (riff.FourCC{'W', 'E', 'B', 'P'}) {
		return ErrInvalidImage
	}
	frames := 0
	for {
		id, _, _, err := chunks.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ErrInvalidImage
		}
		if id == (riff.FourCC{'A', 'N', 'M', 'F'}) {
			frames++
		}
	}
	if frames < 2 {
		return fmt.Errorf("%w: animated webp has %d frames, want at least 2", ErrInvalidManifest, frames)
	}
	return nil
}

func memberError(name string, err error) error {
	if errors.Is(err, ErrBundleTooLarge) {
		return ErrBundleTooLarge
	}
	return fmt.Errorf("%w: %s: %w", ErrInvalidManifest, name, err)
}

// copyBudget caps the total bytes written into a staged version across every
// object, thumbnail and pack thumbnail. The bounded writer stops at the cap
// instead of staging an unbounded work artifact first.
type copyBudget struct {
	remaining int64
	exceeded  bool
}

func newCopyBudget(limit int64) *copyBudget {
	return &copyBudget{remaining: limit}
}

type budgetWriter struct {
	dst    io.Writer
	budget *copyBudget
}

func (w *budgetWriter) Write(p []byte) (int, error) {
	if w.budget.remaining <= 0 {
		w.budget.exceeded = true
		return 0, ErrBundleTooLarge
	}
	if int64(len(p)) > w.budget.remaining {
		allowed := int(w.budget.remaining)
		n, err := w.dst.Write(p[:allowed])
		w.budget.remaining -= int64(n)
		w.budget.exceeded = true
		if err != nil {
			return n, err
		}
		return n, ErrBundleTooLarge
	}
	n, err := w.dst.Write(p)
	w.budget.remaining -= int64(n)
	return n, err
}

func copyFile(dst, src string, extra io.Writer, budget *copyBudget) (string, int64, error) {
	in, err := openNoFollow(src)
	if err != nil {
		return "", 0, err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", 0, fmt.Errorf("assets: create %s: %w", dst, err)
	}
	digest := sha256.New()
	writers := []io.Writer{&budgetWriter{dst: out, budget: budget}, digest}
	if extra != nil {
		writers = append(writers, extra)
	}
	n, copyErr := io.Copy(io.MultiWriter(writers...), io.LimitReader(in, budget.remaining+1))
	if copyErr == nil {
		copyErr = out.Sync()
	}
	if err := out.Close(); copyErr == nil && err != nil {
		copyErr = err
	}
	switch {
	case budget.exceeded || errors.Is(copyErr, ErrBundleTooLarge):
		return "", n, ErrBundleTooLarge
	case copyErr != nil:
		return "", n, fmt.Errorf("assets: copy %s: %w", src, copyErr)
	}
	return hex.EncodeToString(digest.Sum(nil)), n, nil
}

func writeBundle(packDir, bundlePath string, manifest PackManifest) (int64, error) {
	out, err := os.OpenFile(bundlePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, fmt.Errorf("assets: create bundle: %w", err)
	}
	limited := &limitWriter{dst: out, remaining: MaxBundleBytes}
	buffered := bufio.NewWriter(limited)
	writer := zip.NewWriter(buffered)

	members := []string{"pack.json", "thumb.webp"}
	for _, sticker := range manifest.Stickers {
		members = append(members, sticker.File, path.Join("thumbs", sticker.ThumbFile))
		if sticker.Animated != nil {
			members = append(members, path.Join("animated", sticker.Animated.File))
		}
	}
	var writeErr error
	for _, member := range members {
		header := &zip.FileHeader{Name: member, Method: zip.Store}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			writeErr = err
			break
		}
		src, err := os.Open(filepath.Join(packDir, filepath.FromSlash(member)))
		if err != nil {
			writeErr = err
			break
		}
		_, writeErr = io.Copy(entry, src)
		src.Close()
		if writeErr != nil {
			break
		}
	}
	if closeErr := writer.Close(); writeErr == nil && closeErr != nil {
		writeErr = closeErr
	}
	if flushErr := buffered.Flush(); writeErr == nil && flushErr != nil {
		writeErr = flushErr
	}
	if syncErr := out.Sync(); writeErr == nil && syncErr != nil {
		writeErr = syncErr
	}
	if closeErr := out.Close(); writeErr == nil && closeErr != nil {
		writeErr = closeErr
	}
	if errors.Is(writeErr, ErrBundleTooLarge) || limited.exceeded {
		return 0, ErrBundleTooLarge
	}
	if writeErr != nil {
		return 0, fmt.Errorf("assets: write bundle: %w", writeErr)
	}
	return limited.written, nil
}

type limitWriter struct {
	dst       io.Writer
	remaining int64
	written   int64
	exceeded  bool
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		w.exceeded = true
		return 0, ErrBundleTooLarge
	}
	n, err := w.dst.Write(p)
	w.remaining -= int64(n)
	w.written += int64(n)
	return n, err
}
