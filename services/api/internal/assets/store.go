// Package assets owns the local data-root layout: source images, job work
// directories, committed pack versions and bundles. Every filesystem path is
// derived from a server-generated id and kept under the data directory.
package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

const (
	// MaxSourceBytes is the largest accepted input sheet.
	MaxSourceBytes = 8 << 20
	// MaxImageDimension is the largest accepted image edge.
	MaxImageDimension = 4096
	// MaxBundleBytes is the largest accepted output ZIP.
	MaxBundleBytes = 32 << 20
	// MaxStickers is the largest accepted sticker count.
	MaxStickers = 64
	// MaxClipBytes is the largest accepted animation clip upload.
	MaxClipBytes = 20 << 20
)

var (
	ErrInvalidImage     = errors.New("assets: unsupported or invalid image")
	ErrInvalidFilename  = errors.New("assets: invalid filename")
	ErrImageTooLarge    = errors.New("assets: image exceeds 8 MiB")
	ErrImageDimensions  = errors.New("assets: image dimension exceeds 4096 px")
	ErrAnimatedImage    = errors.New("assets: animated images are not accepted")
	ErrInvalidSourceID  = errors.New("assets: invalid source id")
	ErrInvalidJobID     = errors.New("assets: invalid job id")
	ErrUnsafePath       = errors.New("assets: path escapes the data directory")
	ErrNotFound         = errors.New("assets: file not found")
	ErrInvalidManifest  = errors.New("assets: invalid pack manifest")
	ErrBundleTooLarge   = errors.New("assets: bundle exceeds 32 MiB")
	ErrAlreadyCommitted = errors.New("assets: version is already committed")
	ErrInvalidClip      = errors.New("assets: unsupported or invalid clip")
	ErrClipTooLarge     = errors.New("assets: clip exceeds 20 MiB")
	ErrInvalidClipID    = errors.New("assets: invalid clip id")
)

// clipExts are the stored clip extensions, one per sniffed container.
var clipExts = []string{"mp4", "mov", "webm"}

var (
	idPattern        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	jobIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
	pngSignature     = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	committedPattern = regexp.MustCompile(`^(?:packs/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/v[1-9][0-9]*/[A-Za-z0-9][A-Za-z0-9._-]*(?:/[A-Za-z0-9][A-Za-z0-9._-]*)*|bundles/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/v[1-9][0-9]*\.zip)$`)
)

// Source is a persisted upload. RelativePath is always below the data root.
type Source struct {
	ID             string
	Filename       string
	MediaType      string
	ByteSize       int64
	Width          int
	Height         int
	ChecksumSHA256 string
	RelativePath   string
}

// Clip is a persisted animation clip. RelativePath is clips/<ID>.<ext>; the
// extension comes from the sniffed container, never from the filename.
type Clip struct {
	ID             string
	Filename       string
	MediaType      string
	ByteSize       int64
	ChecksumSHA256 string
	RelativePath   string
}

// Store maps the data root to the layout in spec §8.1.
type Store struct {
	dataDir string
}

// NewStore prepares the data root and the top-level resource directories.
func NewStore(dataDir string) (*Store, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, errors.New("assets: data dir is required")
	}
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("assets: resolve data dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("assets: create data dir: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("assets: resolve data dir: %w", err)
	}
	abs = resolved
	for _, sub := range []string{"sources", "work", "packs", "bundles", "clips", "animations"} {
		if err := os.MkdirAll(filepath.Join(abs, sub), 0o755); err != nil {
			return nil, fmt.Errorf("assets: create %s: %w", sub, err)
		}
	}
	return &Store{dataDir: abs}, nil
}

// SaveSource validates one PNG/JPEG/WebP sheet and stores it under a
// server-generated id. The returned RelativePath never contains user input.
func (s *Store) SaveSource(ctx context.Context, r io.Reader, filename string) (Source, error) {
	if err := ctx.Err(); err != nil {
		return Source{}, err
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxSourceBytes+1))
	if err != nil {
		return Source{}, fmt.Errorf("assets: read source: %w", err)
	}
	if len(data) > MaxSourceBytes {
		return Source{}, ErrImageTooLarge
	}
	mediaType, ext, err := sniffImage(data)
	if err != nil {
		return Source{}, err
	}
	if animated(mediaType, data) {
		return Source{}, ErrAnimatedImage
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Source{}, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	if format != formatName(mediaType) {
		return Source{}, fmt.Errorf("%w: media type mismatch", ErrInvalidImage)
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		return Source{}, fmt.Errorf("%w: empty image", ErrInvalidImage)
	}
	if cfg.Width > MaxImageDimension || cfg.Height > MaxImageDimension {
		return Source{}, ErrImageDimensions
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return Source{}, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	name, err := cleanFilename(filename)
	if err != nil {
		return Source{}, err
	}

	id := uuid.NewString()
	dir, err := s.path("sources", id)
	if err != nil {
		return Source{}, err
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return Source{}, fmt.Errorf("assets: create source dir: %w", err)
	}
	relative := path.Join("sources", id, "original."+ext)
	final := filepath.Join(dir, "original."+ext)
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(dir)
		}
	}()
	if err := writeFileSync(final, data); err != nil {
		return Source{}, err
	}
	if err := syncDir(dir); err != nil {
		return Source{}, err
	}
	if err := syncDir(filepath.Join(s.dataDir, "sources")); err != nil {
		return Source{}, err
	}
	committed = true
	sum := sha256.Sum256(data)
	return Source{
		ID:             id,
		Filename:       name,
		MediaType:      mediaType,
		ByteSize:       int64(len(data)),
		Width:          cfg.Width,
		Height:         cfg.Height,
		ChecksumSHA256: hex.EncodeToString(sum[:]),
		RelativePath:   relative,
	}, nil
}

// SourcePath returns the absolute path of a persisted source. Only server ids
// are accepted; symlinks are never followed.
func (s *Store) SourcePath(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", ErrInvalidSourceID
	}
	for _, ext := range []string{"png", "jpg", "webp"} {
		p, err := s.path("sources", id, "original."+ext)
		if err != nil {
			return "", err
		}
		info, err := os.Lstat(p)
		if err == nil && info.Mode().IsRegular() {
			if err := s.verifyParent(p); err != nil {
				return "", err
			}
			return p, nil
		}
	}
	return "", ErrNotFound
}

// OpenSource opens a persisted source for read-only access.
func (s *Store) OpenSource(id string) (*os.File, error) {
	p, err := s.SourcePath(id)
	if err != nil {
		return nil, err
	}
	return openNoFollow(p)
}

// RemoveSource deletes a persisted source directory. A missing one is not an error.
func (s *Store) RemoveSource(id string) error {
	if !idPattern.MatchString(id) {
		return ErrInvalidSourceID
	}
	dir, err := s.path("sources", id)
	if err != nil {
		return err
	}
	if err := s.verifyDir(dir); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return os.RemoveAll(dir)
}

// SaveClip validates one mp4/mov/webm clip by its container magic and stores
// it under a server-generated id. Frames are only decoded later by the
// processor; this check keeps arbitrary files out of the clip queue.
func (s *Store) SaveClip(ctx context.Context, r io.Reader, filename string) (Clip, error) {
	if err := ctx.Err(); err != nil {
		return Clip{}, err
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxClipBytes+1))
	if err != nil {
		return Clip{}, fmt.Errorf("assets: read clip: %w", err)
	}
	if len(data) > MaxClipBytes {
		return Clip{}, ErrClipTooLarge
	}
	mediaType, ext, err := sniffClip(data)
	if err != nil {
		return Clip{}, err
	}
	name, err := cleanFilename(filename)
	if err != nil {
		return Clip{}, err
	}

	id := uuid.NewString()
	dir, err := s.path("clips")
	if err != nil {
		return Clip{}, err
	}
	tmp := filepath.Join(dir, "."+id+".tmp")
	final := filepath.Join(dir, id+"."+ext)
	if err := writeFileSync(tmp, data); err != nil {
		_ = os.Remove(tmp)
		return Clip{}, err
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return Clip{}, fmt.Errorf("assets: commit clip: %w", err)
	}
	if err := syncDir(dir); err != nil {
		_ = os.Remove(final)
		return Clip{}, err
	}
	sum := sha256.Sum256(data)
	return Clip{
		ID:             id,
		Filename:       name,
		MediaType:      mediaType,
		ByteSize:       int64(len(data)),
		ChecksumSHA256: hex.EncodeToString(sum[:]),
		RelativePath:   path.Join("clips", id+"."+ext),
	}, nil
}

// ClipPath returns the absolute path of a persisted clip. Only server ids are
// accepted; symlinks are never followed.
func (s *Store) ClipPath(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", ErrInvalidClipID
	}
	for _, ext := range clipExts {
		p, err := s.path("clips", id+"."+ext)
		if err != nil {
			return "", err
		}
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			return "", ErrUnsafePath
		}
		if err := s.verifyParent(p); err != nil {
			return "", err
		}
		return p, nil
	}
	return "", ErrNotFound
}

// RemoveClip deletes a persisted clip. A missing one is not an error.
func (s *Store) RemoveClip(id string) error {
	if !idPattern.MatchString(id) {
		return ErrInvalidClipID
	}
	for _, ext := range clipExts {
		if err := s.removeFile("clips", id+"."+ext); err != nil {
			return err
		}
	}
	return nil
}

// RemoveAnimated deletes the draft animated WebP animations/<id>.webp of an
// animation. A missing one is not an error.
func (s *Store) RemoveAnimated(id string) error {
	if !idPattern.MatchString(id) {
		return ErrInvalidClipID
	}
	return s.removeFile("animations", id+".webp")
}

// SaveAnimated atomically writes the draft animated WebP animations/<id>.webp
// and returns that data-root-relative path.
func (s *Store) SaveAnimated(id string, data []byte) (string, error) {
	if !idPattern.MatchString(id) {
		return "", ErrInvalidClipID
	}
	final, err := s.path("animations", id+".webp")
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(final)
	if err := s.verifyDir(dir); err != nil {
		return "", err
	}
	tmp := filepath.Join(dir, "."+id+".webp.tmp")
	// A temp file left by a crash would make the exclusive create fail.
	if err := os.Remove(tmp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("assets: clear animated temp: %w", err)
	}
	if err := writeFileSync(tmp, data); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("assets: commit animated: %w", err)
	}
	if err := syncDir(dir); err != nil {
		_ = os.Remove(final)
		return "", err
	}
	return path.Join("animations", id+".webp"), nil
}

// OpenAnimated opens a draft animated WebP by the data-root-relative path
// SaveAnimated returned, without following a symlink.
func (s *Store) OpenAnimated(relativePath string) (*os.File, error) {
	dir, name := path.Split(relativePath)
	id := strings.TrimSuffix(name, ".webp")
	if dir != "animations/" || id == name || !idPattern.MatchString(id) {
		return nil, ErrUnsafePath
	}
	p, err := s.path("animations", name)
	if err != nil {
		return nil, err
	}
	if err := s.verifyParent(p); err != nil {
		return nil, err
	}
	return openNoFollow(p)
}

// PruneDrafts removes the regular files directly inside clips/ and
// animations/ that keep rejects. Symlinks, subdirectories and every other
// directory are never touched, and a symlinked draft directory is refused.
func (s *Store) PruneDrafts(keep func(dir, name string) bool) error {
	var errs []error
	for _, sub := range []string{"clips", "animations"} {
		dir, err := s.path(sub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		info, err := os.Lstat(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("assets: inspect %s: %w", sub, err))
			continue
		}
		if !info.IsDir() {
			errs = append(errs, fmt.Errorf("assets: %s: %w", sub, ErrUnsafePath))
			continue
		}
		if err := s.verifyDir(dir); err != nil {
			errs = append(errs, err)
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("assets: list %s: %w", sub, err))
			continue
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() || keep(sub, entry.Name()) {
				continue
			}
			// os.Remove never follows a symlink swapped in after ReadDir.
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, fmt.Errorf("assets: prune %s/%s: %w", sub, entry.Name(), err))
			}
		}
	}
	return errors.Join(errs...)
}

func (s *Store) removeFile(parts ...string) error {
	p, err := s.path(parts...)
	if err != nil {
		return err
	}
	if err := s.verifyParent(p); err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("assets: remove %s: %w", p, err)
	}
	return nil
}

// WorkDir returns (creating if needed) the per-job directory below work/.
func (s *Store) WorkDir(jobID string) (string, error) {
	if !jobIDPattern.MatchString(jobID) {
		return "", ErrInvalidJobID
	}
	dir, err := s.path("work", jobID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("assets: create work dir: %w", err)
	}
	return dir, nil
}

// OpenCommitted opens a file that belongs to a committed pack version or
// bundle. Callers must still check the database allowlist first; this method
// only proves the relative path is one the store can produce.
func (s *Store) OpenCommitted(relativePath string) (*os.File, error) {
	rel := path.Clean(strings.ReplaceAll(relativePath, "\\", "/"))
	if !committedPattern.MatchString(rel) {
		return nil, ErrUnsafePath
	}
	p, err := s.path(filepath.FromSlash(rel))
	if err != nil {
		return nil, err
	}
	if err := s.verifyParent(p); err != nil {
		return nil, err
	}
	return openNoFollow(p)
}

func (s *Store) path(parts ...string) (string, error) {
	p := filepath.Join(append([]string{s.dataDir}, parts...)...)
	rel, err := filepath.Rel(s.dataDir, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", ErrUnsafePath
	}
	return p, nil
}

// verifyDir rejects a path with a symlinked component that leaves the data
// root; the processor shares our UID, so it could otherwise place one there.
func (s *Store) verifyDir(dir string) error {
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("assets: resolve %s: %w", dir, err)
	}
	if !s.contains(resolved) {
		return ErrUnsafePath
	}
	return nil
}

func (s *Store) verifyParent(p string) error {
	return s.verifyDir(filepath.Dir(p))
}

func (s *Store) contains(p string) bool {
	rel, err := filepath.Rel(s.dataDir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func sniffImage(data []byte) (mediaType, ext string, err error) {
	switch {
	case len(data) >= len(pngSignature) && bytes.Equal(data[:len(pngSignature)], pngSignature):
		return "image/png", "png", nil
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg", "jpg", nil
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp", "webp", nil
	default:
		return "", "", ErrInvalidImage
	}
}

// sniffClip recognises ISO BMFF (an ftyp box at offset 4; brand "qt  " is
// QuickTime) and EBML (WebM/Matroska) containers.
func sniffClip(data []byte) (mediaType, ext string, err error) {
	switch {
	case len(data) >= 12 && string(data[4:8]) == "ftyp" && string(data[8:12]) == "qt  ":
		return "video/quicktime", "mov", nil
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		return "video/mp4", "mp4", nil
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}):
		return "video/webm", "webm", nil
	default:
		return "", "", ErrInvalidClip
	}
}

func formatName(mediaType string) string {
	switch mediaType {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpeg"
	case "image/webp":
		return "webp"
	default:
		return ""
	}
}

// animated detects APNG and animated WebP containers before decoding; neither
// decoder reports animation on its own.
func animated(mediaType string, data []byte) bool {
	switch mediaType {
	case "image/png":
		return pngAnimated(data)
	case "image/webp":
		return webpAnimated(data)
	default:
		return false
	}
}

func pngAnimated(data []byte) bool {
	pos := len(pngSignature)
	for pos+12 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		kind := string(data[pos+4 : pos+8])
		if kind == "acTL" {
			return true
		}
		if kind == "IEND" || length < 0 || pos+12+length > len(data) {
			return false
		}
		pos += 12 + length
	}
	return false
}

func webpAnimated(data []byte) bool {
	pos := 12
	for pos+8 <= len(data) {
		kind := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		payload := pos + 8
		switch kind {
		case "ANIM", "ANMF":
			return true
		case "VP8X":
			if payload < len(data) && data[payload]&0x02 != 0 {
				return true
			}
		}
		next := payload + size + (size & 1)
		if size < 0 || next <= pos || next > len(data) {
			return false
		}
		pos = next
	}
	return false
}

func cleanFilename(name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = path.Base(name)
	if name == "" || name == "." || name == "/" || name == ".." ||
		strings.ContainsRune(name, 0) || len(name) > 200 {
		return "", ErrInvalidFilename
	}
	return name, nil
}

func openNoFollow(p string) (*os.File, error) {
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		var errno syscall.Errno
		if errors.As(err, &errno) && (errno == syscall.ELOOP || errno == syscall.ENOTDIR) {
			return nil, ErrUnsafePath
		}
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, ErrNotFound
	}
	return f, nil
}

func writeFileSync(p string, data []byte) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("assets: create %s: %w", p, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("assets: write %s: %w", p, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("assets: fsync %s: %w", p, err)
	}
	return f.Close()
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("assets: open dir %s: %w", dir, err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("assets: fsync dir %s: %w", dir, err)
	}
	return nil
}
