// Package studio owns the reusable fragment library, the character profiles and
// the sticker sheet designs that compose the prompt an operator copies into an
// external image tool. No image or text model is called from here.
package studio

import "time"

// Scope records the layer a fragment belongs to: fixed identity, one pack, or a
// single sticker cell.
type Scope string

const (
	ScopeIdentity Scope = "identity"
	ScopePack     Scope = "pack"
	ScopeSticker  Scope = "sticker"
)

// Colors holds the four palette roles of a palette fragment.
type Colors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
	Line      string `json:"line"`
}

// Group is a named bucket of fragments ("D01 species and body").
type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Fragment is one reusable idea: a Vietnamese name for the operator, an English
// prompt phrase for the image tool. Palette fragments also carry Colors.
type Fragment struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	Name      string    `json:"name"`
	NameEn    string    `json:"name_en"`
	Prompt    string    `json:"prompt"`
	Notes     string    `json:"notes"`
	Source    string    `json:"source"`
	Scope     Scope     `json:"scope"`
	Colors    *Colors   `json:"colors,omitempty"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FragmentDraft is the writable part of a fragment.
type FragmentDraft struct {
	GroupID string  `json:"group_id"`
	Name    string  `json:"name"`
	NameEn  string  `json:"name_en"`
	Prompt  string  `json:"prompt"`
	Notes   string  `json:"notes"`
	Source  string  `json:"source"`
	Scope   Scope   `json:"scope"`
	Colors  *Colors `json:"colors,omitempty"`
}

// GroupDraft is the writable part of a group.
type GroupDraft struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// ReferenceAsset is an admin-only uploaded image; the bytes live beside the
// pack sources under sources/<id>/original.<ext>.
type ReferenceAsset struct {
	ID             string    `json:"id"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	ByteSize       int64     `json:"byte_size"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	CreatedAt      time.Time `json:"created_at"`
}

// Character is a reusable identity: what must stay the same across packs.
type Character struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	IdentityNotes    string    `json:"identity_notes"`
	RightsNotes      string    `json:"rights_notes"`
	ReferenceAssetID string    `json:"reference_asset_id,omitempty"`
	Archived         bool      `json:"archived"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CharacterDraft is the writable part of a character.
type CharacterDraft struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	IdentityNotes    string `json:"identity_notes"`
	RightsNotes      string `json:"rights_notes"`
	ReferenceAssetID string `json:"reference_asset_id"`
}

// Cell describes one sticker of the sheet.
type Cell struct {
	Action      string   `json:"action"`
	Expression  string   `json:"expression"`
	FragmentIDs []string `json:"fragment_ids,omitempty"`
	FreeText    string   `json:"free_text"`
}

// Draft is the editable composition. IDs reference the library; free text is
// always kept verbatim. An empty OutlineColor means the #FFFFFF default, while
// OutlineWidth is a plain int: a new-draft caller (UI or HTTP) preinitialises it
// with DefaultOutlineWidth, and an explicit 0 means no outline.
type Draft struct {
	ID                string   `json:"id,omitempty"`
	Name              string   `json:"name"`
	CharacterID       string   `json:"character_id,omitempty"`
	GroupIDs          []string `json:"group_ids,omitempty"`
	FragmentIDs       []string `json:"fragment_ids,omitempty"`
	PinnedFragmentIDs []string `json:"pinned_fragment_ids,omitempty"`
	PaletteID         string   `json:"palette_id,omitempty"`
	FreeText          string   `json:"free_text,omitempty"`
	Rows              int      `json:"rows"`
	Columns           int      `json:"columns"`
	Cells             []Cell   `json:"cells"`
	OutlineWidth      int      `json:"outline_width"`
	OutlineColor      string   `json:"outline_color"`
}

// Selection is one fragment resolved at compile time: its text is frozen into
// the snapshot so later library edits never rewrite a saved prompt.
type Selection struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	FragmentID string `json:"fragment_id"`
	Name       string `json:"name"`
	Prompt     string `json:"prompt"`
	Scope      Scope  `json:"scope"`
}

// SnapshotCharacter is the frozen identity block of a compiled design.
type SnapshotCharacter struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	IdentityNotes string `json:"identity_notes"`
}

// SnapshotPalette is the frozen palette block of a compiled design.
type SnapshotPalette struct {
	FragmentID string `json:"fragment_id"`
	Name       string `json:"name"`
	Prompt     string `json:"prompt"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Line       string `json:"line"`
}

// SnapshotCell is one frozen sticker cell.
type SnapshotCell struct {
	Row        int         `json:"row"`
	Column     int         `json:"column"`
	Action     string      `json:"action"`
	Expression string      `json:"expression"`
	FreeText   string      `json:"free_text"`
	Fragments  []Selection `json:"fragments"`
}

// Snapshot is the persisted description of a compiled design, stored with
// schema_version 1 in designs.snapshot_json.
type Snapshot struct {
	SchemaVersion int                `json:"schema_version"`
	Character     *SnapshotCharacter `json:"character,omitempty"`
	Selections    []Selection        `json:"selections"`
	Palette       *SnapshotPalette   `json:"palette,omitempty"`
	FreeText      string             `json:"free_text"`
	Rows          int                `json:"rows"`
	Columns       int                `json:"columns"`
	OutlineWidth  int                `json:"outline_width"`
	OutlineColor  string             `json:"outline_color"`
	Cells         []SnapshotCell     `json:"cells"`
	CompiledAt    string             `json:"compiled_at"`
}

// CompiledPrompt is the copyable prompt plus the snapshot it was built from.
type CompiledPrompt struct {
	Text     string
	Snapshot Snapshot
}

// Design is a saved composition: its draft, the frozen snapshot and the prompt
// text generated at save time.
type Design struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CharacterID  string    `json:"character_id,omitempty"`
	Rows         int       `json:"rows"`
	Columns      int       `json:"columns"`
	OutlineWidth int       `json:"outline_width"`
	OutlineColor string    `json:"outline_color"`
	Draft        Draft     `json:"draft"`
	Snapshot     Snapshot  `json:"snapshot"`
	Prompt       string    `json:"compiled_prompt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FragmentFilter narrows a library listing.
type FragmentFilter struct {
	Query           string
	GroupID         string
	Scope           Scope
	IncludeArchived bool
	Page            int
	PageSize        int
}

// FragmentPage is one page of fragments ordered by group then name.
type FragmentPage struct {
	Items    []Fragment `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

// DesignFilter narrows a design listing.
type DesignFilter struct {
	Query    string
	Page     int
	PageSize int
}

// DesignPage is one page of designs ordered by updated_at then id.
type DesignPage struct {
	Items    []Design `json:"items"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Total    int      `json:"total"`
}

// MaxSheetsPerDesign caps the generated sheets kept for one design.
const MaxSheetsPerDesign = 20

// Sheet is one externally generated sticker sheet image for a design, with the
// provider that made it and the prompt the design had at upload time.
type Sheet struct {
	ID             string    `json:"id"`
	DesignID       string    `json:"design_id"`
	Provider       string    `json:"provider"`
	Note           string    `json:"note"`
	PromptSnapshot string    `json:"prompt_snapshot"`
	Filename       string    `json:"filename"`
	MediaType      string    `json:"media_type"`
	ByteSize       int64     `json:"byte_size"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	CreatedAt      time.Time `json:"created_at"`
}
