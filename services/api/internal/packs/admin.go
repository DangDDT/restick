package packs

import (
	"context"
	"fmt"
	"strings"
)

// AdminFilter selects packs in any lifecycle state for the owner console. Page
// starts at 1; PageSize defaults to 20 and is capped at 100.
type AdminFilter struct {
	Query    string
	Status   string
	Tag      string
	Page     int
	PageSize int
}

// ListAll returns packs of every status ordered by updated_at descending, then
// id. Unlike ListReady it never widens public visibility: callers must be owner
// sessions.
func (s *Service) ListAll(ctx context.Context, filter AdminFilter) (Page, error) {
	page, size := normalizePage(filter.Page, filter.PageSize)
	status := strings.TrimSpace(filter.Status)
	switch Status(status) {
	case "", StatusDraft, StatusReady, StatusRevoked:
	default:
		return Page{}, fmt.Errorf("%w: unknown status", ErrInvalid)
	}

	where := []string{"1 = 1"}
	args := []any{}
	if status != "" {
		where = append(where, "p.status = ?")
		args = append(args, status)
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		pattern := "%" + query + "%"
		where = append(where, "(p.name LIKE ? OR p.slug LIKE ? OR p.author LIKE ? OR p.description LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if tag := strings.TrimSpace(filter.Tag); tag != "" {
		where = append(where, "EXISTS (SELECT 1 FROM pack_tags t WHERE t.pack_id = p.id AND t.tag = ?)")
		args = append(args, tag)
	}
	clause := " WHERE " + strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM packs p`+clause, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("packs: count all: %w", err)
	}

	query := `SELECT ` + packColumns + packFrom + clause +
		` ORDER BY p.updated_at DESC, p.id ASC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), size, int64(page-1)*int64(size))
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return Page{}, fmt.Errorf("packs: list all: %w", err)
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
