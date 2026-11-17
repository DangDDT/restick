// Package packs owns the catalog records: pack metadata, tags, versions and
// immutable sticker metadata. File bytes and manifests belong to the assets
// package.
package packs

import (
	"errors"
	"time"
)

// Status is the pack lifecycle state.
type Status string

const (
	StatusDraft   Status = "draft"
	StatusReady   Status = "ready"
	StatusRevoked Status = "revoked"
)

var (
	ErrNotFound        = errors.New("packs: pack not found")
	ErrVersionNotFound = errors.New("packs: version not found")
	ErrSlugTaken       = errors.New("packs: slug already reserved")
	ErrJobActive       = errors.New("packs: pack has an active job")
	ErrNotDraft        = errors.New("packs: only draft packs can be deleted")
	ErrRevoked         = errors.New("packs: pack is revoked")
	ErrInvalid         = errors.New("packs: invalid pack")
)

// Metadata is the mutable catalog part of a pack. It never affects committed
// version bytes or their manifest snapshot.
type Metadata struct {
	Name        string
	Author      string
	Description string
	Tags        []string
	DesignID    string
	SheetID     string

	// ThemeColor is #RRGGBB; nil keeps the stored colour, "" clears it.
	ThemeColor *string
}

// PackDraft creates a new pack. Slug is permanent and case-insensitively
// unique for the lifetime of the row, including after revoke.
type PackDraft struct {
	Slug        string
	Name        string
	Author      string
	Description string
	Tags        []string
	DesignID    string
	SheetID     string
}

// PackSheet is the design sheet a pack is cut from. ID is also the sources/
// directory name, never a client supplied path.
type PackSheet struct {
	ID             string
	Provider       string
	Filename       string
	ChecksumSHA256 string
	Width          int
	Height         int
	ByteSize       int64
}

// Pack is the catalog view. Summary fields refer to the current committed
// version; paths are data-root relative.
type Pack struct {
	ID             string
	Slug           string
	Name           string
	Author         string
	Description    string
	Tags           []string
	DesignID       string
	Status         Status
	CurrentVersion int
	StickerCount   int
	ChecksumSHA256 string
	ThumbnailPath  string
	SheetID        string
	Sheet          *PackSheet

	// ThemeColor is the admin override; ThemeColorSuggested is derived from the
	// stickers. Both are "" or #RRGGBB.
	ThemeColor          string
	ThemeColorSuggested string
	// Previews are the first stickers of the current version (at most 6).
	Previews  []PreviewSticker
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PreviewSticker is a data-root relative sticker file pair for marketplace cards.
type PreviewSticker struct {
	ThumbPath, ObjectPath string
}

// EffectiveThemeColor is the admin colour, else the suggested one, else "".
func (p Pack) EffectiveThemeColor() string {
	if p.ThemeColor != "" {
		return p.ThemeColor
	}
	return p.ThemeColorSuggested
}

// Filter selects ready packs. Page starts at 1; PageSize defaults to 20 and is
// capped at 100.
type Filter struct {
	Tag      string
	KeyID    string // when set, only packs granted to this API key
	Page     int
	PageSize int
}

// Page is one stable page ordered by updated_at then id.
type Page struct {
	Items    []Pack
	Page     int
	PageSize int
	Total    int
}

// Version is an immutable committed version snapshot.
type Version struct {
	PackID         string
	Version        int
	JobID          string
	ChecksumSHA256 string
	StickerCount   int
	ManifestJSON   []byte
	BundlePath     string
	BundleByteSize int64
	ThumbnailPath  string
	OutputDir      string
	CreatedAt      time.Time
	Stickers       []Sticker
}

// Sticker is one immutable sticker row; both paths are data-root relative.
type Sticker struct {
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
