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
	"unicode/utf8"
)

var (
	ErrSheetLimit      = errors.New("studio: design already has the maximum number of sheets")
	ErrSheetInUse      = errors.New("studio: sheet is used by a pack")
	ErrDesignHasSheets = errors.New("studio: design still has sheets")
	providerPattern    = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)
)

const (
	maxSheetNoteLength  = 500
	sheetColumns        = `id, design_id, provider, note, prompt_snapshot, filename, media_type, byte_size, width, height, checksum_sha256, created_at`
	errSheetUniqueText  = "UNIQUE constraint failed"
	errSheetForeignText = "FOREIGN KEY constraint failed"
)

// SaveSheet stores one generated image for a design. The same bytes uploaded
// twice for a design return the existing sheet.
func (s *Service) SaveSheet(ctx context.Context, designID string, r io.Reader, filename, provider, note string) (Sheet, error) {
	if s.files == nil {
		return Sheet{}, ErrFilesUnavailable
	}
	provider = strings.TrimSpace(provider)
	note = strings.TrimSpace(note)
	if !providerPattern.MatchString(provider) || utf8.RuneCountInString(note) > maxSheetNoteLength {
		return Sheet{}, ErrInvalid
	}
	design, err := s.GetDesign(ctx, designID)
	if err != nil {
		return Sheet{}, err
	}
	source, err := s.files.SaveSource(ctx, r, filename)
	if err != nil {
		return Sheet{}, err
	}
	// Identical bytes win over the limit check: return the existing sheet.
	if existing, err := s.sheetByChecksum(ctx, designID, source.ChecksumSHA256); err == nil {
		_ = s.files.RemoveSource(source.ID)
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		_ = s.files.RemoveSource(source.ID)
		return Sheet{}, fmt.Errorf("studio: find duplicate sheet: %w", err)
	}
	// The count guard in the INSERT keeps concurrent uploads under the limit.
	result, err := s.db.ExecContext(ctx, `INSERT INTO design_sheets
		(id, design_id, provider, note, prompt_snapshot, filename, media_type, byte_size, width, height, checksum_sha256)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		WHERE (SELECT COUNT(*) FROM design_sheets WHERE design_id = ?) < ?`,
		source.ID, designID, provider, note, design.Prompt, source.Filename, source.MediaType, source.ByteSize,
		source.Width, source.Height, source.ChecksumSHA256, designID, MaxSheetsPerDesign)
	if err == nil {
		var affected int64
		if affected, err = result.RowsAffected(); err == nil && affected == 0 {
			_ = s.files.RemoveSource(source.ID)
			if winner, err := s.sheetByChecksum(ctx, designID, source.ChecksumSHA256); err == nil {
				return winner, nil
			} else if !errors.Is(err, sql.ErrNoRows) {
				return Sheet{}, fmt.Errorf("studio: find duplicate sheet: %w", err)
			}
			return Sheet{}, ErrSheetLimit
		}
	}
	if err != nil {
		_ = s.files.RemoveSource(source.ID)
		if strings.Contains(err.Error(), errSheetUniqueText) {
			winner, err := s.sheetByChecksum(ctx, designID, source.ChecksumSHA256)
			if err != nil {
				return Sheet{}, fmt.Errorf("studio: find duplicate sheet: %w", err)
			}
			return winner, nil
		}
		return Sheet{}, fmt.Errorf("studio: insert sheet: %w", err)
	}
	return s.GetSheet(ctx, designID, source.ID)
}

// ListSheets returns the sheets of a design, newest first.
func (s *Service) ListSheets(ctx context.Context, designID string) ([]Sheet, error) {
	if _, err := s.GetDesign(ctx, designID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+sheetColumns+` FROM design_sheets
		WHERE design_id = ? ORDER BY created_at DESC, rowid DESC`, designID)
	if err != nil {
		return nil, fmt.Errorf("studio: list sheets: %w", err)
	}
	defer rows.Close()
	items := []Sheet{}
	for rows.Next() {
		item, err := scanSheet(rows)
		if err != nil {
			return nil, fmt.Errorf("studio: scan sheet: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("studio: read sheets: %w", err)
	}
	return items, nil
}

// GetSheet returns one sheet that belongs to the design.
func (s *Service) GetSheet(ctx context.Context, designID, sheetID string) (Sheet, error) {
	item, err := scanSheet(s.db.QueryRowContext(ctx, `SELECT `+sheetColumns+` FROM design_sheets
		WHERE id = ? AND design_id = ?`, sheetID, designID))
	if errors.Is(err, sql.ErrNoRows) {
		return Sheet{}, ErrNotFound
	}
	if err != nil {
		return Sheet{}, fmt.Errorf("studio: get sheet: %w", err)
	}
	return item, nil
}

// OpenSheet returns the sheet metadata plus a read handle on its image.
func (s *Service) OpenSheet(ctx context.Context, designID, sheetID string) (Sheet, *os.File, error) {
	sheet, err := s.GetSheet(ctx, designID, sheetID)
	if err != nil {
		return Sheet{}, nil, err
	}
	if s.files == nil {
		return Sheet{}, nil, ErrFilesUnavailable
	}
	file, err := s.files.OpenSource(sheet.ID)
	if err != nil {
		return Sheet{}, nil, err
	}
	return sheet, file, nil
}

// DeleteSheet removes a sheet unless a pack still references it.
func (s *Service) DeleteSheet(ctx context.Context, designID, sheetID string) error {
	if s.files == nil {
		return ErrFilesUnavailable
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM design_sheets WHERE id = ? AND design_id = ?`, sheetID, designID)
	if err != nil {
		if strings.Contains(err.Error(), errSheetForeignText) {
			return ErrSheetInUse
		}
		return fmt.Errorf("studio: delete sheet: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		return ErrNotFound
	}
	return s.files.RemoveSource(sheetID)
}

func (s *Service) sheetByChecksum(ctx context.Context, designID, checksum string) (Sheet, error) {
	item, err := scanSheet(s.db.QueryRowContext(ctx, `SELECT `+sheetColumns+` FROM design_sheets
		WHERE design_id = ? AND checksum_sha256 = ?`, designID, checksum))
	if err != nil {
		return Sheet{}, err
	}
	return item, nil
}

func scanSheet(row scanner) (Sheet, error) {
	var item Sheet
	var createdAt string
	if err := row.Scan(&item.ID, &item.DesignID, &item.Provider, &item.Note, &item.PromptSnapshot,
		&item.Filename, &item.MediaType, &item.ByteSize, &item.Width, &item.Height,
		&item.ChecksumSHA256, &createdAt); err != nil {
		return Sheet{}, err
	}
	item.CreatedAt = parseTimestamp(createdAt)
	return item, nil
}
