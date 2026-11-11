package studio

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const designColumns = `id, name, COALESCE(character_id, ''), rows, columns, outline_width, outline_color,
	draft_json, snapshot_json, compiled_prompt, created_at, updated_at`

// SaveDesign compiles the draft and stores it together with the frozen snapshot
// and prompt. A draft with an ID rewrites that design as a new snapshot; the
// previous prompt is not kept as version history.
func (s *Service) SaveDesign(ctx context.Context, draft Draft) (Design, error) {
	draft = normalizeDraft(draft)
	switch {
	case draft.Name == "":
		return Design{}, fmt.Errorf("%w: design name is required", ErrInvalid)
	case len(draft.Name) > 200:
		return Design{}, fmt.Errorf("%w: design name is too long", ErrInvalid)
	}
	compiled, err := s.Compile(ctx, draft)
	if err != nil {
		return Design{}, err
	}
	created := draft.ID == ""
	if created {
		draft.ID = uuid.NewString()
	}
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return Design{}, fmt.Errorf("studio: encode draft: %w", err)
	}
	snapshotJSON, err := json.Marshal(compiled.Snapshot)
	if err != nil {
		return Design{}, fmt.Errorf("studio: encode snapshot: %w", err)
	}

	if created {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO designs
			(id, name, character_id, rows, columns, outline_width, outline_color,
			 draft_json, snapshot_json, compiled_prompt)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			draft.ID, draft.Name, nullable(draft.CharacterID), draft.Rows, draft.Columns,
			draft.OutlineWidth, draft.OutlineColor, string(draftJSON), string(snapshotJSON),
			compiled.Text); err != nil {
			return Design{}, fmt.Errorf("studio: insert design: %w", err)
		}
		return s.GetDesign(ctx, draft.ID)
	}

	result, err := s.db.ExecContext(ctx, `UPDATE designs SET
		name = ?, character_id = ?, rows = ?, columns = ?, outline_width = ?, outline_color = ?,
		draft_json = ?, snapshot_json = ?, compiled_prompt = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`,
		draft.Name, nullable(draft.CharacterID), draft.Rows, draft.Columns,
		draft.OutlineWidth, draft.OutlineColor, string(draftJSON), string(snapshotJSON),
		compiled.Text, draft.ID)
	if err != nil {
		return Design{}, fmt.Errorf("studio: update design: %w", err)
	}
	if err := requireRow(ctx, s.db, result, "designs", draft.ID); err != nil {
		return Design{}, err
	}
	return s.GetDesign(ctx, draft.ID)
}

// GetDesign returns a saved design with its stored draft, snapshot and prompt.
func (s *Service) GetDesign(ctx context.Context, id string) (Design, error) {
	item, err := scanDesign(s.db.QueryRowContext(ctx,
		`SELECT `+designColumns+` FROM designs WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Design{}, ErrNotFound
	}
	if err != nil {
		return Design{}, fmt.Errorf("studio: get design: %w", err)
	}
	return item, nil
}

// ListDesigns returns saved designs ordered by updated_at descending, then id.
func (s *Service) ListDesigns(ctx context.Context, filter DesignFilter) (DesignPage, error) {
	page, size := normalizePage(filter.Page, filter.PageSize)
	where := "1 = 1"
	args := []any{}
	if query := strings.TrimSpace(filter.Query); query != "" {
		where = "name LIKE ?"
		args = append(args, "%"+query+"%")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM designs WHERE `+where, args...).Scan(&total); err != nil {
		return DesignPage{}, fmt.Errorf("studio: count designs: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+designColumns+` FROM designs WHERE `+where+
		` ORDER BY updated_at DESC, id ASC LIMIT ? OFFSET ?`,
		append(args, size, int64(page-1)*int64(size))...)
	if err != nil {
		return DesignPage{}, fmt.Errorf("studio: list designs: %w", err)
	}
	defer rows.Close()
	items := []Design{}
	for rows.Next() {
		item, err := scanDesign(rows)
		if err != nil {
			return DesignPage{}, fmt.Errorf("studio: scan design: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return DesignPage{}, fmt.Errorf("studio: read designs: %w", err)
	}
	return DesignPage{Items: items, Page: page, PageSize: size, Total: total}, nil
}

// DeleteDesign removes the editable record. Packs that referenced it keep their
// own snapshot and have design_id cleared by the foreign key.
func (s *Service) DeleteDesign(ctx context.Context, id string) error {
	var sheets int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM design_sheets WHERE design_id = ?`, id).Scan(&sheets); err != nil {
		return fmt.Errorf("studio: count design sheets: %w", err)
	}
	if sheets > 0 {
		return ErrDesignHasSheets
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM designs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("studio: delete design: %w", err)
	}
	return requireRow(ctx, s.db, result, "designs", id)
}

func scanDesign(row scanner) (Design, error) {
	var (
		item         Design
		characterID  string
		draftJSON    string
		snapshotJSON string
		createdAt    string
		updatedAt    string
	)
	if err := row.Scan(&item.ID, &item.Name, &characterID, &item.Rows, &item.Columns,
		&item.OutlineWidth, &item.OutlineColor, &draftJSON, &snapshotJSON,
		&item.Prompt, &createdAt, &updatedAt); err != nil {
		return Design{}, err
	}
	item.CharacterID = characterID
	if err := json.Unmarshal([]byte(draftJSON), &item.Draft); err != nil {
		return Design{}, fmt.Errorf("studio: decode draft: %w", err)
	}
	if err := json.Unmarshal([]byte(snapshotJSON), &item.Snapshot); err != nil {
		return Design{}, fmt.Errorf("studio: decode snapshot: %w", err)
	}
	item.CreatedAt = parseTimestamp(createdAt)
	item.UpdatedAt = parseTimestamp(updatedAt)
	return item, nil
}
