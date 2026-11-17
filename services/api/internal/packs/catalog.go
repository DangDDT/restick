package packs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const timestampLayout = "2006-01-02T15:04:05Z"

const packColumns = `p.id, p.slug, p.name, p.author, p.description, p.design_id,
	p.status, p.current_version,
	p.sheet_id, s.provider, s.filename, s.checksum_sha256, s.width, s.height, s.byte_size,
	p.created_at, p.updated_at,
	COALESCE(v.sticker_count, 0), COALESCE(v.checksum_sha256, ''), COALESCE(v.thumbnail_path, ''),
	p.theme_color, p.theme_color_suggested`

const packFrom = ` FROM packs p
	LEFT JOIN design_sheets s ON s.id = p.sheet_id
	LEFT JOIN pack_versions v ON v.pack_id = p.id AND v.version = p.current_version`

var (
	slugPattern       = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	checksumPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	themeColorPattern = regexp.MustCompile(`^#[0-9A-F]{6}$`)
)

// Service exposes the catalog and immutable version metadata.
type Service struct {
	db *sql.DB
}

// NewService wraps the single factory database connection.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Create inserts a draft pack and its tags in one transaction.
func (s *Service) Create(ctx context.Context, draft PackDraft) (Pack, error) {
	slug := strings.TrimSpace(draft.Slug)
	if !slugPattern.MatchString(slug) {
		return Pack{}, fmt.Errorf("%w: slug must be lowercase words separated by single dashes", ErrInvalid)
	}
	metadata, err := normalizeMetadata(Metadata{
		Name:        draft.Name,
		Author:      draft.Author,
		Description: draft.Description,
		Tags:        draft.Tags,
		DesignID:    draft.DesignID,
		SheetID:     draft.SheetID,
	})
	if err != nil {
		return Pack{}, err
	}
	id := uuid.NewString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Pack{}, err
	}
	defer tx.Rollback()

	if err := requireSheet(ctx, tx, metadata); err != nil {
		return Pack{}, err
	}
	var exists int
	switch err := tx.QueryRowContext(ctx, `SELECT 1 FROM packs WHERE slug = ?`, slug).Scan(&exists); {
	case err == nil:
		return Pack{}, ErrSlugTaken
	case !errors.Is(err, sql.ErrNoRows):
		return Pack{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO packs (id, slug, name, author, description, design_id, sheet_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, slug, metadata.Name, metadata.Author, metadata.Description, metadata.DesignID, metadata.SheetID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: packs.slug") {
			return Pack{}, ErrSlugTaken
		}
		return Pack{}, fmt.Errorf("packs: insert pack: %w", err)
	}
	if err := replaceTags(ctx, tx, id, metadata.Tags); err != nil {
		return Pack{}, err
	}
	if err := tx.Commit(); err != nil {
		return Pack{}, err
	}
	return s.Get(ctx, id)
}

// UpdateCatalog edits mutable metadata and tags only. The committed manifest
// snapshot, bundle and stickers are never touched.
func (s *Service) UpdateCatalog(ctx context.Context, id string, metadata Metadata) (Pack, error) {
	if !isUUID(id) {
		return Pack{}, ErrNotFound
	}
	normalized, err := normalizeMetadata(metadata)
	if err != nil {
		return Pack{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Pack{}, err
	}
	defer tx.Rollback()

	status, err := packStatus(ctx, tx, id)
	if err != nil {
		return Pack{}, err
	}
	if status == StatusRevoked {
		return Pack{}, ErrRevoked
	}
	if err := requireSheet(ctx, tx, normalized); err != nil {
		return Pack{}, err
	}
	// Changing the sheet mid-run would desync the queued job's source.
	var changed int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM packs WHERE id = ? AND sheet_id IS NOT ?`,
		id, normalized.SheetID).Scan(&changed); err != nil {
		return Pack{}, err
	}
	if changed > 0 {
		active, err := hasActiveJob(ctx, tx, id)
		if err != nil {
			return Pack{}, err
		}
		if active {
			return Pack{}, ErrJobActive
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE packs
		SET name = ?, author = ?, description = ?, design_id = ?, sheet_id = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`,
		normalized.Name, normalized.Author, normalized.Description, normalized.DesignID, normalized.SheetID, id); err != nil {
		return Pack{}, fmt.Errorf("packs: update catalog: %w", err)
	}
	if err := replaceTags(ctx, tx, id, normalized.Tags); err != nil {
		return Pack{}, err
	}
	if normalized.ThemeColor != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE packs SET theme_color = ? WHERE id = ?`, *normalized.ThemeColor, id); err != nil {
			return Pack{}, fmt.Errorf("packs: update theme color: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Pack{}, err
	}
	return s.Get(ctx, id)
}

// requireSheet checks the sheet exists and belongs to the pack's design.
func requireSheet(ctx context.Context, tx *sql.Tx, m Metadata) error {
	var ok int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM design_sheets WHERE id = ? AND design_id = ?`,
		m.SheetID, m.DesignID).Scan(&ok)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: sheet does not belong to the design", ErrInvalid)
	}
	return err
}

// ListReady returns ready packs ordered by updated_at descending, then id.
func (s *Service) ListReady(ctx context.Context, filter Filter) (Page, error) {
	page, size := normalizePage(filter.Page, filter.PageSize)
	tag := strings.TrimSpace(filter.Tag)

	join, args := "", []any{}
	if tag != "" {
		join = ` JOIN pack_tags t ON t.pack_id = p.id AND t.tag = ?`
		args = append(args, tag)
	}
	from := packFrom + join
	countFrom := ` FROM packs p` + join
	where := ` WHERE p.status = 'ready'`
	if filter.KeyID != "" {
		where += ` AND p.id IN (SELECT pack_id FROM api_key_packs WHERE key_id = ?)`
		args = append(args, filter.KeyID)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+countFrom+where, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("packs: count ready: %w", err)
	}

	query := `SELECT ` + packColumns + from + where +
		` ORDER BY p.updated_at DESC, p.id ASC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), size, int64(page-1)*int64(size))
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return Page{}, fmt.Errorf("packs: list ready: %w", err)
	}
	items, err := scanPacks(rows)
	if err != nil {
		return Page{}, err
	}
	if err := loadTags(ctx, s.db, items); err != nil {
		return Page{}, err
	}
	return Page{Items: items, Page: page, PageSize: size, Total: total}, nil
}

// Get returns a pack in any state so callers can map draft/revoked visibility.
func (s *Service) Get(ctx context.Context, id string) (Pack, error) {
	if !isUUID(id) {
		return Pack{}, ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+packColumns+packFrom+` WHERE p.id = ?`, id)
	item, err := scanPack(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Pack{}, ErrNotFound
	}
	if err != nil {
		return Pack{}, fmt.Errorf("packs: get pack: %w", err)
	}
	tags, err := tagsFor(ctx, s.db, id)
	if err != nil {
		return Pack{}, err
	}
	item.Tags = tags
	return item, nil
}

// GetVersion returns one immutable version and its stickers.
func (s *Service) GetVersion(ctx context.Context, id string, version int) (Version, error) {
	if !isUUID(id) || version < 1 {
		return Version{}, ErrVersionNotFound
	}
	var item Version
	var createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT job_id, checksum_sha256, sticker_count, manifest_json,
			bundle_path, bundle_byte_size, thumbnail_path, output_dir, created_at
		FROM pack_versions WHERE pack_id = ? AND version = ?`, id, version).Scan(
		&item.JobID, &item.ChecksumSHA256, &item.StickerCount, &item.ManifestJSON,
		&item.BundlePath, &item.BundleByteSize, &item.ThumbnailPath, &item.OutputDir, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM packs WHERE id = ?`, id).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Version{}, ErrNotFound
			}
			return Version{}, fmt.Errorf("packs: get version: %w", err)
		}
		return Version{}, ErrVersionNotFound
	}
	if err != nil {
		return Version{}, fmt.Errorf("packs: get version: %w", err)
	}
	item.PackID = id
	item.Version = version
	item.CreatedAt = parseTimestamp(createdAt)

	rows, err := s.db.QueryContext(ctx, `SELECT sort_order, filename, object_path, thumb_path, checksum_sha256,
			width, height, emoji_hint, keywords_json, animated_path, animated_checksum_sha256,
			animated_byte_size, animated_frames, animated_duration_ms
		FROM stickers WHERE pack_id = ? AND version = ? ORDER BY sort_order`, id, version)
	if err != nil {
		return Version{}, fmt.Errorf("packs: get stickers: %w", err)
	}
	defer rows.Close()
	item.Stickers = []Sticker{}
	for rows.Next() {
		var sticker Sticker
		var keywords string
		if err := rows.Scan(&sticker.SortOrder, &sticker.Filename, &sticker.ObjectPath, &sticker.ThumbPath,
			&sticker.ChecksumSHA256, &sticker.Width, &sticker.Height, &sticker.EmojiHint, &keywords,
			&sticker.AnimatedPath, &sticker.AnimatedChecksumSHA256, &sticker.AnimatedByteSize,
			&sticker.AnimatedFrames, &sticker.AnimatedDurationMS); err != nil {
			return Version{}, fmt.Errorf("packs: scan sticker: %w", err)
		}
		if err := json.Unmarshal([]byte(keywords), &sticker.Keywords); err != nil {
			return Version{}, fmt.Errorf("packs: decode keywords: %w", err)
		}
		item.Stickers = append(item.Stickers, sticker)
	}
	if err := rows.Err(); err != nil {
		return Version{}, fmt.Errorf("packs: read stickers: %w", err)
	}
	return item, nil
}

// DeleteDraft removes an unused draft. Ready and revoked packs are kept so
// slugs and tombstones stay valid.
func (s *Service) DeleteDraft(ctx context.Context, id string) error {
	if !isUUID(id) {
		return ErrNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	status, err := packStatus(ctx, tx, id)
	if err != nil {
		return err
	}
	if status != StatusDraft {
		return ErrNotDraft
	}
	active, err := hasActiveJob(ctx, tx, id)
	if err != nil {
		return err
	}
	if active {
		return ErrJobActive
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM packs WHERE id = ? AND status = 'draft'`, id); err != nil {
		return fmt.Errorf("packs: delete draft: %w", err)
	}
	return tx.Commit()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPack(row scanner) (Pack, error) {
	var (
		item          Pack
		designID      sql.NullString
		sheetID       sql.NullString
		provider      sql.NullString
		sheetFile     sql.NullString
		sheetSum      sql.NullString
		sheetWidth    sql.NullInt64
		sheetHeight   sql.NullInt64
		sheetSize     sql.NullInt64
		status        string
		createdAtText string
		updatedAtText string
	)
	if err := row.Scan(&item.ID, &item.Slug, &item.Name, &item.Author, &item.Description, &designID,
		&status, &item.CurrentVersion,
		&sheetID, &provider, &sheetFile, &sheetSum, &sheetWidth, &sheetHeight, &sheetSize,
		&createdAtText, &updatedAtText,
		&item.StickerCount, &item.ChecksumSHA256, &item.ThumbnailPath,
		&item.ThemeColor, &item.ThemeColorSuggested); err != nil {
		return Pack{}, err
	}
	item.DesignID = designID.String
	item.Status = Status(status)
	item.CreatedAt = parseTimestamp(createdAtText)
	item.UpdatedAt = parseTimestamp(updatedAtText)
	item.SheetID = sheetID.String
	if sheetID.Valid && provider.Valid {
		item.Sheet = &PackSheet{
			ID:             sheetID.String,
			Provider:       provider.String,
			Filename:       sheetFile.String,
			ChecksumSHA256: sheetSum.String,
			Width:          int(sheetWidth.Int64),
			Height:         int(sheetHeight.Int64),
			ByteSize:       sheetSize.Int64,
		}
	}
	return item, nil
}

func scanPacks(rows *sql.Rows) ([]Pack, error) {
	defer rows.Close()
	items := []Pack{}
	for rows.Next() {
		item, err := scanPack(rows)
		if err != nil {
			return nil, fmt.Errorf("packs: scan pack: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("packs: read packs: %w", err)
	}
	return items, nil
}

func loadTags(ctx context.Context, db *sql.DB, items []Pack) error {
	if len(items) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(items)), ",")
	args := make([]any, len(items))
	for i, item := range items {
		args[i] = item.ID
	}
	rows, err := db.QueryContext(ctx, `SELECT pack_id, tag FROM pack_tags
		WHERE pack_id IN (`+placeholders+`) ORDER BY tag`, args...)
	if err != nil {
		return fmt.Errorf("packs: list tags: %w", err)
	}
	defer rows.Close()
	byPack := map[string][]string{}
	for rows.Next() {
		var packID, tag string
		if err := rows.Scan(&packID, &tag); err != nil {
			return fmt.Errorf("packs: scan tag: %w", err)
		}
		byPack[packID] = append(byPack[packID], tag)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("packs: read tags: %w", err)
	}
	for i := range items {
		items[i].Tags = byPack[items[i].ID]
		if items[i].Tags == nil {
			items[i].Tags = []string{}
		}
	}
	return nil
}

func tagsFor(ctx context.Context, db *sql.DB, packID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT tag FROM pack_tags WHERE pack_id = ? ORDER BY tag`, packID)
	if err != nil {
		return nil, fmt.Errorf("packs: list tags: %w", err)
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("packs: scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func packStatus(ctx context.Context, tx *sql.Tx, id string) (Status, error) {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM packs WHERE id = ?`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("packs: read status: %w", err)
	}
	return Status(status), nil
}

func hasActiveJob(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM processing_jobs
		WHERE pack_id = ? AND status IN ('queued', 'running') LIMIT 1`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("packs: check active job: %w", err)
	}
	return true, nil
}

func replaceTags(ctx context.Context, tx *sql.Tx, packID string, tags []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM pack_tags WHERE pack_id = ?`, packID); err != nil {
		return fmt.Errorf("packs: clear tags: %w", err)
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pack_tags (pack_id, tag) VALUES (?, ?)`, packID, tag); err != nil {
			return fmt.Errorf("packs: insert tag: %w", err)
		}
	}
	return nil
}

func normalizeMetadata(metadata Metadata) (Metadata, error) {
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Author = strings.TrimSpace(metadata.Author)
	metadata.Description = strings.TrimSpace(metadata.Description)
	metadata.DesignID = strings.TrimSpace(metadata.DesignID)
	metadata.SheetID = strings.TrimSpace(metadata.SheetID)
	if metadata.ThemeColor != nil {
		color := strings.ToUpper(strings.TrimSpace(*metadata.ThemeColor))
		if color != "" && !themeColorPattern.MatchString(color) {
			return Metadata{}, fmt.Errorf("%w: theme_color must be #RRGGBB", ErrInvalid)
		}
		metadata.ThemeColor = &color
	}
	switch {
	case metadata.Name == "" || metadata.Author == "":
		return Metadata{}, fmt.Errorf("%w: name and author are required", ErrInvalid)
	case len(metadata.Name) > 200 || len(metadata.Author) > 200 || len(metadata.Description) > 4000:
		return Metadata{}, fmt.Errorf("%w: metadata is too long", ErrInvalid)
	case !isUUID(metadata.DesignID):
		return Metadata{}, fmt.Errorf("%w: a design is required", ErrInvalid)
	case !isUUID(metadata.SheetID):
		return Metadata{}, fmt.Errorf("%w: a design sheet is required", ErrInvalid)
	}
	tags := make([]string, 0, len(metadata.Tags))
	seen := make(map[string]struct{}, len(metadata.Tags))
	for _, raw := range metadata.Tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if len(tag) > 40 {
			return Metadata{}, fmt.Errorf("%w: tag is too long", ErrInvalid)
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	if len(tags) > 20 {
		return Metadata{}, fmt.Errorf("%w: too many tags", ErrInvalid)
	}
	metadata.Tags = tags
	return metadata, nil
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

func isUUID(value string) bool {
	return uuid.Validate(value) == nil
}

func parseTimestamp(value string) time.Time {
	parsed, err := time.Parse(timestampLayout, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
