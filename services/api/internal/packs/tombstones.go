package packs

import (
	"context"
	"fmt"
	"time"
)

// Tombstone is one revoked pack as needed by the public delta-sync endpoint.
type Tombstone struct {
	PackID         string
	RevokedAt      time.Time
	ReasonCode     string
	ChecksumSHA256 string
}

// Tombstones returns every tombstone revoked at or after since, ordered by
// (revoked_at, id). Timestamps compare as the canonical UTC second text, so
// the boundary is inclusive at second precision. There is deliberately no
// pagination: delta clients must never miss a record sharing the same second.
func (s *Service) Tombstones(ctx context.Context, since time.Time) ([]Tombstone, error) {
	sinceText := since.UTC().Format("2006-01-02T15:04:05Z")
	rows, err := s.db.QueryContext(ctx, `SELECT pack_id, revoked_at, reason_code, checksum_sha256
		FROM pack_tombstones WHERE revoked_at >= ? ORDER BY revoked_at, id`, sinceText)
	if err != nil {
		return nil, fmt.Errorf("packs: list tombstones: %w", err)
	}
	defer rows.Close()
	items := []Tombstone{}
	for rows.Next() {
		var item Tombstone
		var revokedAt string
		if err := rows.Scan(&item.PackID, &revokedAt, &item.ReasonCode, &item.ChecksumSHA256); err != nil {
			return nil, fmt.Errorf("packs: scan tombstone: %w", err)
		}
		item.RevokedAt = parseTimestamp(revokedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("packs: read tombstones: %w", err)
	}
	return items, nil
}
