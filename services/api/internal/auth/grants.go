package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrUnknownPack is returned when a grant names a pack that does not exist.
var ErrUnknownPack = errors.New("auth: unknown pack id")

// Removal records that a pack was taken away from a key, so clients can drop it.
type Removal struct {
	PackID    string
	RemovedAt time.Time
}

// KeyPacks returns the sorted pack ids granted to a key.
func (s *Service) KeyPacks(ctx context.Context, keyID string) ([]string, error) {
	var one int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM api_keys WHERE id = ?`, keyID).Scan(&one); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrKeyNotFound
	} else if err != nil {
		return nil, fmt.Errorf("auth: key packs: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT pack_id FROM api_key_packs WHERE key_id = ? ORDER BY pack_id`, keyID)
	if err != nil {
		return nil, fmt.Errorf("auth: key packs: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("auth: key packs: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SetKeyPacks replaces a key's grants atomically. Dropped packs get a removal
// marker; re-granted packs lose theirs.
func (s *Service) SetKeyPacks(ctx context.Context, keyID string, packIDs []string) error {
	want := map[string]bool{}
	for _, id := range packIDs {
		want[id] = true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("auth: set key packs: %w", err)
	}
	defer tx.Rollback()

	var one int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM api_keys WHERE id = ?`, keyID).Scan(&one); errors.Is(err, sql.ErrNoRows) {
		return ErrKeyNotFound
	} else if err != nil {
		return fmt.Errorf("auth: set key packs: %w", err)
	}
	if len(want) > 0 {
		args := make([]any, 0, len(want))
		for id := range want {
			args = append(args, id)
		}
		var found int
		q := `SELECT COUNT(*) FROM packs WHERE id IN (?` + strings.Repeat(",?", len(args)-1) + `)`
		if err := tx.QueryRowContext(ctx, q, args...).Scan(&found); err != nil {
			return fmt.Errorf("auth: set key packs: %w", err)
		}
		if found != len(want) {
			return ErrUnknownPack
		}
	}

	rows, err := tx.QueryContext(ctx, `SELECT pack_id FROM api_key_packs WHERE key_id = ?`, keyID)
	if err != nil {
		return fmt.Errorf("auth: set key packs: %w", err)
	}
	var current []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("auth: set key packs: %w", err)
		}
		current = append(current, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("auth: set key packs: %w", err)
	}

	now := utcSeconds(time.Now())
	have := map[string]bool{}
	for _, id := range current {
		have[id] = true
		if want[id] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM api_key_packs WHERE key_id = ? AND pack_id = ?`, keyID, id); err != nil {
			return fmt.Errorf("auth: set key packs: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO api_key_pack_removals (key_id, pack_id, removed_at) VALUES (?, ?, ?)`, keyID, id, now); err != nil {
			return fmt.Errorf("auth: set key packs: %w", err)
		}
	}
	for id := range want {
		if have[id] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO api_key_packs (key_id, pack_id) VALUES (?, ?)`, keyID, id); err != nil {
			return fmt.Errorf("auth: set key packs: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM api_key_pack_removals WHERE key_id = ? AND pack_id = ?`, keyID, id); err != nil {
			return fmt.Errorf("auth: set key packs: %w", err)
		}
	}
	return tx.Commit()
}

// KeyHasPack reports whether the key is granted the pack.
func (s *Service) KeyHasPack(ctx context.Context, keyID, packID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM api_key_packs WHERE key_id = ? AND pack_id = ?`, keyID, packID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("auth: key has pack: %w", err)
	}
	return true, nil
}

// KeyRemovals lists removal markers at or after since, oldest first.
func (s *Service) KeyRemovals(ctx context.Context, keyID string, since time.Time) ([]Removal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pack_id, removed_at FROM api_key_pack_removals WHERE key_id = ? AND removed_at >= ? ORDER BY removed_at, pack_id`,
		keyID, utcSeconds(since))
	if err != nil {
		return nil, fmt.Errorf("auth: key removals: %w", err)
	}
	defer rows.Close()
	items := []Removal{}
	for rows.Next() {
		var (
			r  Removal
			at string
		)
		if err := rows.Scan(&r.PackID, &at); err != nil {
			return nil, fmt.Errorf("auth: key removals: %w", err)
		}
		r.RemovedAt = parseSeconds(at)
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth: key removals: %w", err)
	}
	return items, nil
}
