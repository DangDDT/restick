package studio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/google/uuid"
)

const timestampLayout = "2006-01-02T15:04:05Z"

// Errors returned to the admin HTTP layer; codes are mapped there.
var (
	ErrNotFound         = errors.New("studio: not found")
	ErrInvalid          = errors.New("studio: invalid input")
	ErrGroupNotEmpty    = errors.New("studio: group still has fragments")
	ErrFilesUnavailable = errors.New("studio: file store is not configured")
)

var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Service exposes the library, characters and designs on the single factory
// database connection.
type Service struct {
	db    *sql.DB
	files *assets.Store
}

// NewService wires the studio to the database and the data-root file store.
// files may be nil for compile-only callers; reference uploads then fail.
func NewService(db *sql.DB, files *assets.Store) *Service {
	return &Service{db: db, files: files}
}

const groupColumns = `id, name, description, sort_order, created_at, updated_at`

// CreateGroup inserts a group and returns it.
func (s *Service) CreateGroup(ctx context.Context, draft GroupDraft) (Group, error) {
	normalized, err := normalizeGroup(draft)
	if err != nil {
		return Group{}, err
	}
	id := uuid.NewString()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO fragment_groups (id, name, description, sort_order)
		VALUES (?, ?, ?, ?)`, id, normalized.Name, normalized.Description, normalized.SortOrder); err != nil {
		return Group{}, fmt.Errorf("studio: insert group: %w", err)
	}
	return s.GetGroup(ctx, id)
}

// UpdateGroup rewrites the mutable group fields.
func (s *Service) UpdateGroup(ctx context.Context, id string, draft GroupDraft) (Group, error) {
	normalized, err := normalizeGroup(draft)
	if err != nil {
		return Group{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE fragment_groups
		SET name = ?, description = ?, sort_order = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`, normalized.Name, normalized.Description, normalized.SortOrder, id)
	if err != nil {
		return Group{}, fmt.Errorf("studio: update group: %w", err)
	}
	if err := requireRow(ctx, s.db, result, "fragment_groups", id); err != nil {
		return Group{}, err
	}
	return s.GetGroup(ctx, id)
}

// DeleteGroup removes an empty group. A group that still holds fragments, even
// archived ones, is refused so no fragment loses its bucket.
func (s *Service) DeleteGroup(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var members int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM fragments WHERE group_id = ?`, id).Scan(&members); err != nil {
		return fmt.Errorf("studio: count group fragments: %w", err)
	}
	if members > 0 {
		return ErrGroupNotEmpty
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM fragment_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("studio: delete group: %w", err)
	}
	if err := requireRow(ctx, tx, result, "fragment_groups", id); err != nil {
		return err
	}
	return tx.Commit()
}

// GetGroup returns one group.
func (s *Service) GetGroup(ctx context.Context, id string) (Group, error) {
	item, err := scanGroup(s.db.QueryRowContext(ctx, `SELECT `+groupColumns+` FROM fragment_groups WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, fmt.Errorf("studio: get group: %w", err)
	}
	return item, nil
}

// ListGroups returns every group in display order.
func (s *Service) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+groupColumns+` FROM fragment_groups ORDER BY sort_order, name, id`)
	if err != nil {
		return nil, fmt.Errorf("studio: list groups: %w", err)
	}
	defer rows.Close()
	items := []Group{}
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("studio: scan group: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const fragmentColumns = `f.id, f.group_id, f.name, f.name_en, f.prompt, f.notes, f.source, f.scope,
	COALESCE(f.primary_color, ''), COALESCE(f.secondary_color, ''), COALESCE(f.accent_color, ''), COALESCE(f.line_color, ''),
	f.archived_at IS NOT NULL, f.created_at, f.updated_at`

// CreateFragment inserts a fragment and returns it.
func (s *Service) CreateFragment(ctx context.Context, draft FragmentDraft) (Fragment, error) {
	normalized, err := normalizeFragment(draft)
	if err != nil {
		return Fragment{}, err
	}
	if err := s.requireGroup(ctx, normalized.GroupID); err != nil {
		return Fragment{}, err
	}
	id := uuid.NewString()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO fragments
		(id, group_id, name, name_en, prompt, notes, source, scope,
		 primary_color, secondary_color, accent_color, line_color)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, normalized.GroupID, normalized.Name, normalized.NameEn, normalized.Prompt,
		normalized.Notes, normalized.Source, string(normalized.Scope),
		colorValue(normalized.Colors, 0), colorValue(normalized.Colors, 1),
		colorValue(normalized.Colors, 2), colorValue(normalized.Colors, 3)); err != nil {
		return Fragment{}, fmt.Errorf("studio: insert fragment: %w", err)
	}
	return s.GetFragment(ctx, id)
}

// UpdateFragment rewrites a fragment. The archive flag is not touched here so a
// historical selection keeps its meaning until it is archived deliberately.
func (s *Service) UpdateFragment(ctx context.Context, id string, draft FragmentDraft) (Fragment, error) {
	normalized, err := normalizeFragment(draft)
	if err != nil {
		return Fragment{}, err
	}
	if err := s.requireGroup(ctx, normalized.GroupID); err != nil {
		return Fragment{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE fragments SET
		group_id = ?, name = ?, name_en = ?, prompt = ?, notes = ?, source = ?, scope = ?,
		primary_color = ?, secondary_color = ?, accent_color = ?, line_color = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`,
		normalized.GroupID, normalized.Name, normalized.NameEn, normalized.Prompt,
		normalized.Notes, normalized.Source, string(normalized.Scope),
		colorValue(normalized.Colors, 0), colorValue(normalized.Colors, 1),
		colorValue(normalized.Colors, 2), colorValue(normalized.Colors, 3), id)
	if err != nil {
		return Fragment{}, fmt.Errorf("studio: update fragment: %w", err)
	}
	if err := requireRow(ctx, s.db, result, "fragments", id); err != nil {
		return Fragment{}, err
	}
	return s.GetFragment(ctx, id)
}

// ArchiveFragment hides a fragment from new selections. Rows are kept so saved
// snapshots and older designs still resolve.
func (s *Service) ArchiveFragment(ctx context.Context, id string) (Fragment, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE fragments
		SET archived_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND archived_at IS NULL`, id)
	if err != nil {
		return Fragment{}, fmt.Errorf("studio: archive fragment: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Fragment{}, err
	}
	if affected == 0 {
		// Already archived is idempotent; a missing row is not.
		if _, err := s.GetFragment(ctx, id); err != nil {
			return Fragment{}, err
		}
	}
	return s.GetFragment(ctx, id)
}

// GetFragment returns one fragment, archived or not.
func (s *Service) GetFragment(ctx context.Context, id string) (Fragment, error) {
	item, err := scanFragment(s.db.QueryRowContext(ctx,
		`SELECT `+fragmentColumns+` FROM fragments f WHERE f.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Fragment{}, ErrNotFound
	}
	if err != nil {
		return Fragment{}, fmt.Errorf("studio: get fragment: %w", err)
	}
	return item, nil
}

// ListFragments searches Vietnamese/English names and filters group and scope.
func (s *Service) ListFragments(ctx context.Context, filter FragmentFilter) (FragmentPage, error) {
	page, size := normalizePage(filter.Page, filter.PageSize)
	where := []string{"1 = 1"}
	args := []any{}
	if query := strings.TrimSpace(filter.Query); query != "" {
		where = append(where, `(f.name LIKE ? OR f.name_en LIKE ?)`)
		pattern := "%" + query + "%"
		args = append(args, pattern, pattern)
	}
	if filter.GroupID != "" {
		where = append(where, `f.group_id = ?`)
		args = append(args, filter.GroupID)
	}
	if filter.Scope != "" {
		if !validScope(filter.Scope) {
			return FragmentPage{}, fmt.Errorf("%w: unknown scope %q", ErrInvalid, filter.Scope)
		}
		where = append(where, `f.scope = ?`)
		args = append(args, string(filter.Scope))
	}
	if !filter.IncludeArchived {
		where = append(where, `f.archived_at IS NULL`)
	}
	clause := ` FROM fragments f JOIN fragment_groups g ON g.id = f.group_id WHERE ` + strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+clause, args...).Scan(&total); err != nil {
		return FragmentPage{}, fmt.Errorf("studio: count fragments: %w", err)
	}
	query := `SELECT ` + fragmentColumns + clause +
		` ORDER BY g.sort_order, g.name, f.name, f.id LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, query, append(args, size, int64(page-1)*int64(size))...)
	if err != nil {
		return FragmentPage{}, fmt.Errorf("studio: list fragments: %w", err)
	}
	defer rows.Close()
	items := []Fragment{}
	for rows.Next() {
		item, err := scanFragment(rows)
		if err != nil {
			return FragmentPage{}, fmt.Errorf("studio: scan fragment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return FragmentPage{}, fmt.Errorf("studio: read fragments: %w", err)
	}
	return FragmentPage{Items: items, Page: page, PageSize: size, Total: total}, nil
}

// SaveReferenceAsset validates and stores one admin-only image, then records it.
func (s *Service) SaveReferenceAsset(ctx context.Context, r io.Reader, filename string) (ReferenceAsset, error) {
	if s.files == nil {
		return ReferenceAsset{}, ErrFilesUnavailable
	}
	source, err := s.files.SaveSource(ctx, r, filename)
	if err != nil {
		return ReferenceAsset{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO reference_assets
		(id, filename, content_type, byte_size, width, height, checksum_sha256)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		source.ID, source.Filename, source.MediaType, source.ByteSize,
		source.Width, source.Height, source.ChecksumSHA256); err != nil {
		return ReferenceAsset{}, fmt.Errorf("studio: insert reference asset: %w", err)
	}
	return s.GetReferenceAsset(ctx, source.ID)
}

// GetReferenceAsset returns the metadata of one private image.
func (s *Service) GetReferenceAsset(ctx context.Context, id string) (ReferenceAsset, error) {
	var item ReferenceAsset
	var createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT id, filename, content_type, byte_size, width, height,
			checksum_sha256, created_at
		FROM reference_assets WHERE id = ?`, id).Scan(&item.ID, &item.Filename, &item.ContentType,
		&item.ByteSize, &item.Width, &item.Height, &item.ChecksumSHA256, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReferenceAsset{}, ErrNotFound
	}
	if err != nil {
		return ReferenceAsset{}, fmt.Errorf("studio: get reference asset: %w", err)
	}
	item.CreatedAt = parseTimestamp(createdAt)
	return item, nil
}

// OpenReferenceAsset returns metadata plus a read handle on the private bytes.
func (s *Service) OpenReferenceAsset(ctx context.Context, id string) (ReferenceAsset, *os.File, error) {
	asset, err := s.GetReferenceAsset(ctx, id)
	if err != nil {
		return ReferenceAsset{}, nil, err
	}
	if s.files == nil {
		return ReferenceAsset{}, nil, ErrFilesUnavailable
	}
	file, err := s.files.OpenSource(asset.ID)
	if err != nil {
		return ReferenceAsset{}, nil, err
	}
	return asset, file, nil
}

const characterColumns = `id, name, description, identity_notes, rights_notes,
	archived_at IS NOT NULL, created_at, updated_at`

// CreateCharacter inserts a character profile and its ordered references.
func (s *Service) CreateCharacter(ctx context.Context, draft CharacterDraft) (Character, error) {
	normalized, err := s.normalizeCharacter(ctx, draft)
	if err != nil {
		return Character{}, err
	}
	id := uuid.NewString()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Character{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO characters
		(id, name, description, identity_notes, rights_notes)
		VALUES (?, ?, ?, ?, ?)`,
		id, normalized.Name, normalized.Description, normalized.IdentityNotes,
		normalized.RightsNotes); err != nil {
		return Character{}, fmt.Errorf("studio: insert character: %w", err)
	}
	if err := insertCharacterReferences(ctx, tx, id, normalized.ReferenceAssetIDs); err != nil {
		return Character{}, err
	}
	if err := tx.Commit(); err != nil {
		return Character{}, fmt.Errorf("studio: insert character: %w", err)
	}
	return s.GetCharacter(ctx, id)
}

// UpdateCharacter rewrites a character profile and replaces its reference list
// in one transaction.
func (s *Service) UpdateCharacter(ctx context.Context, id string, draft CharacterDraft) (Character, error) {
	normalized, err := s.normalizeCharacter(ctx, draft)
	if err != nil {
		return Character{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Character{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE characters SET
		name = ?, description = ?, identity_notes = ?, rights_notes = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`,
		normalized.Name, normalized.Description, normalized.IdentityNotes,
		normalized.RightsNotes, id)
	if err != nil {
		return Character{}, fmt.Errorf("studio: update character: %w", err)
	}
	if err := requireRow(ctx, tx, result, "characters", id); err != nil {
		return Character{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM character_reference_assets WHERE character_id = ?`, id); err != nil {
		return Character{}, fmt.Errorf("studio: clear character references: %w", err)
	}
	if err := insertCharacterReferences(ctx, tx, id, normalized.ReferenceAssetIDs); err != nil {
		return Character{}, err
	}
	if err := tx.Commit(); err != nil {
		return Character{}, fmt.Errorf("studio: update character: %w", err)
	}
	return s.GetCharacter(ctx, id)
}

// insertCharacterReferences records the ordered association rows.
func insertCharacterReferences(ctx context.Context, tx *sql.Tx, characterID string, assetIDs []string) error {
	for position, assetID := range assetIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO character_reference_assets
			(character_id, asset_id, position) VALUES (?, ?, ?)`,
			characterID, assetID, position); err != nil {
			return fmt.Errorf("studio: insert character reference: %w", err)
		}
	}
	return nil
}

// ArchiveCharacter hides a character from new drafts; saved designs keep the
// character id and their own snapshot.
func (s *Service) ArchiveCharacter(ctx context.Context, id string) (Character, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE characters
		SET archived_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ? AND archived_at IS NULL`, id); err != nil {
		return Character{}, fmt.Errorf("studio: archive character: %w", err)
	}
	return s.GetCharacter(ctx, id)
}

// GetCharacter returns one character, archived or not.
func (s *Service) GetCharacter(ctx context.Context, id string) (Character, error) {
	item, err := scanCharacter(s.db.QueryRowContext(ctx,
		`SELECT `+characterColumns+` FROM characters WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Character{}, ErrNotFound
	}
	if err != nil {
		return Character{}, fmt.Errorf("studio: get character: %w", err)
	}
	if item.ReferenceAssetIDs, err = s.characterReferenceIDs(ctx, id); err != nil {
		return Character{}, err
	}
	return item, nil
}

// characterReferenceIDs returns one character's ordered reference assets.
func (s *Service) characterReferenceIDs(ctx context.Context, id string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT asset_id FROM character_reference_assets
		WHERE character_id = ? ORDER BY position`, id)
	if err != nil {
		return nil, fmt.Errorf("studio: get character references: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			return nil, fmt.Errorf("studio: scan character reference: %w", err)
		}
		ids = append(ids, assetID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("studio: read character references: %w", err)
	}
	return ids, nil
}

// ListCharacters returns the active characters in name order.
func (s *Service) ListCharacters(ctx context.Context) ([]Character, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+characterColumns+` FROM characters WHERE archived_at IS NULL ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("studio: list characters: %w", err)
	}
	items := []Character{}
	for rows.Next() {
		item, err := scanCharacter(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("studio: scan character: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("studio: list characters: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("studio: list characters: %w", err)
	}
	if err := s.attachCharacterReferences(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

// attachCharacterReferences loads every list in one query after the character
// rows are closed, because the factory database has a single connection.
func (s *Service) attachCharacterReferences(ctx context.Context, items []Character) error {
	wanted := make(map[string]bool, len(items))
	for index := range items {
		items[index].ReferenceAssetIDs = []string{}
		wanted[items[index].ID] = true
	}
	if len(items) == 0 {
		return nil
	}
	// Unparameterized so an arbitrary number of listed characters cannot hit
	// SQLite's host-parameter limit; listing has no page cap.
	rows, err := s.db.QueryContext(ctx, `SELECT character_id, asset_id FROM character_reference_assets
		ORDER BY character_id, position`)
	if err != nil {
		return fmt.Errorf("studio: list character references: %w", err)
	}
	byCharacter := make(map[string][]string, len(items))
	for rows.Next() {
		var characterID, assetID string
		if err := rows.Scan(&characterID, &assetID); err != nil {
			rows.Close()
			return fmt.Errorf("studio: scan character reference: %w", err)
		}
		if wanted[characterID] {
			byCharacter[characterID] = append(byCharacter[characterID], assetID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("studio: list character references: %w", err)
	}
	rows.Close()
	for index, item := range items {
		if ids, ok := byCharacter[item.ID]; ok {
			items[index].ReferenceAssetIDs = ids
		}
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanGroup(row scanner) (Group, error) {
	var item Group
	var createdAt, updatedAt string
	if err := row.Scan(&item.ID, &item.Name, &item.Description, &item.SortOrder, &createdAt, &updatedAt); err != nil {
		return Group{}, err
	}
	item.CreatedAt = parseTimestamp(createdAt)
	item.UpdatedAt = parseTimestamp(updatedAt)
	return item, nil
}

func scanFragment(row scanner) (Fragment, error) {
	var (
		item                       Fragment
		scope                      string
		primary, secondary, accent string
		line                       string
		archived                   bool
		createdAt, updatedAt       string
	)
	if err := row.Scan(&item.ID, &item.GroupID, &item.Name, &item.NameEn, &item.Prompt, &item.Notes,
		&item.Source, &scope, &primary, &secondary, &accent, &line, &archived, &createdAt, &updatedAt); err != nil {
		return Fragment{}, err
	}
	item.Scope = Scope(scope)
	item.Archived = archived
	item.CreatedAt = parseTimestamp(createdAt)
	item.UpdatedAt = parseTimestamp(updatedAt)
	if primary != "" || secondary != "" || accent != "" || line != "" {
		item.Colors = &Colors{Primary: primary, Secondary: secondary, Accent: accent, Line: line}
	}
	return item, nil
}

func scanCharacter(row scanner) (Character, error) {
	var item Character
	var archived bool
	var createdAt, updatedAt string
	if err := row.Scan(&item.ID, &item.Name, &item.Description, &item.IdentityNotes, &item.RightsNotes,
		&archived, &createdAt, &updatedAt); err != nil {
		return Character{}, err
	}
	item.Archived = archived
	item.CreatedAt = parseTimestamp(createdAt)
	item.UpdatedAt = parseTimestamp(updatedAt)
	return item, nil
}

func normalizeGroup(draft GroupDraft) (GroupDraft, error) {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Description = strings.TrimSpace(draft.Description)
	switch {
	case draft.Name == "":
		return GroupDraft{}, fmt.Errorf("%w: group name is required", ErrInvalid)
	case len(draft.Name) > 200 || len(draft.Description) > 4000:
		return GroupDraft{}, fmt.Errorf("%w: group text is too long", ErrInvalid)
	}
	return draft, nil
}

func normalizeFragment(draft FragmentDraft) (FragmentDraft, error) {
	draft.GroupID = strings.TrimSpace(draft.GroupID)
	draft.Name = strings.TrimSpace(draft.Name)
	draft.NameEn = strings.TrimSpace(draft.NameEn)
	draft.Prompt = strings.TrimSpace(draft.Prompt)
	draft.Notes = strings.TrimSpace(draft.Notes)
	draft.Source = strings.TrimSpace(draft.Source)
	switch {
	case draft.GroupID == "":
		return FragmentDraft{}, fmt.Errorf("%w: group is required", ErrInvalid)
	case draft.Name == "":
		return FragmentDraft{}, fmt.Errorf("%w: fragment name is required", ErrInvalid)
	case draft.Prompt == "":
		return FragmentDraft{}, fmt.Errorf("%w: fragment prompt is required", ErrInvalid)
	case len(draft.Name) > 200 || len(draft.NameEn) > 200 || len(draft.Prompt) > 2000 ||
		len(draft.Notes) > 2000 || len(draft.Source) > 500:
		return FragmentDraft{}, fmt.Errorf("%w: fragment text is too long", ErrInvalid)
	case !validScope(draft.Scope):
		return FragmentDraft{}, fmt.Errorf("%w: unknown scope %q", ErrInvalid, draft.Scope)
	}
	if draft.Colors != nil {
		for _, color := range []string{draft.Colors.Primary, draft.Colors.Secondary, draft.Colors.Accent, draft.Colors.Line} {
			if !colorPattern.MatchString(color) {
				return FragmentDraft{}, fmt.Errorf("%w: palette colours must be #RRGGBB", ErrInvalid)
			}
		}
	}
	return draft, nil
}

func (s *Service) normalizeCharacter(ctx context.Context, draft CharacterDraft) (CharacterDraft, error) {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Description = strings.TrimSpace(draft.Description)
	draft.IdentityNotes = strings.TrimSpace(draft.IdentityNotes)
	draft.RightsNotes = strings.TrimSpace(draft.RightsNotes)
	draft.ReferenceAssetIDs = uniqueStrings(draft.ReferenceAssetIDs)
	switch {
	case draft.Name == "":
		return CharacterDraft{}, fmt.Errorf("%w: character name is required", ErrInvalid)
	case len(draft.Name) > 200 || len(draft.Description) > 4000 ||
		len(draft.IdentityNotes) > 4000 || len(draft.RightsNotes) > 4000:
		return CharacterDraft{}, fmt.Errorf("%w: character text is too long", ErrInvalid)
	}
	for _, assetID := range draft.ReferenceAssetIDs {
		if _, err := s.GetReferenceAsset(ctx, assetID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return CharacterDraft{}, fmt.Errorf("%w: unknown reference asset", ErrInvalid)
			}
			return CharacterDraft{}, err
		}
	}
	return draft, nil
}

func (s *Service) requireGroup(ctx context.Context, id string) error {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM fragment_groups WHERE id = ?`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: unknown group %q", ErrInvalid, id)
	}
	if err != nil {
		return fmt.Errorf("studio: check group: %w", err)
	}
	return nil
}

func requireRow(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, result sql.Result, table, id string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	var one int
	err = db.QueryRowContext(ctx, `SELECT 1 FROM `+table+` WHERE id = ?`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("studio: check %s: %w", table, err)
	}
	return nil
}

func colorValue(colors *Colors, index int) any {
	if colors == nil {
		return nil
	}
	switch index {
	case 0:
		return colors.Primary
	case 1:
		return colors.Secondary
	case 2:
		return colors.Accent
	default:
		return colors.Line
	}
}

func validScope(scope Scope) bool {
	switch scope {
	case ScopeIdentity, ScopePack, ScopeSticker:
		return true
	}
	return false
}

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func parseTimestamp(value string) time.Time {
	parsed, err := time.Parse(timestampLayout, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
