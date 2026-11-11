package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// apiKeyPrefix is the readable part of a raw key, shown in key listings.
const apiKeyPrefix = "rsk_"

// CreatedKey carries the one-time raw key returned by create and rotate.
type CreatedKey struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
	Raw    string `json:"raw"`
}

// CreateKey mints a read-only client key. Only its SHA256 digest, prefix and
// optional expiry are stored; Raw is returned exactly once.
func (s *Service) CreateKey(ctx context.Context, name string, expiresAt *time.Time) (CreatedKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CreatedKey{}, ErrEmptyKeyName
	}
	key, err := newKey(name, expiresAt)
	if err != nil {
		return CreatedKey{}, err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, name, prefix, key_hash, active, created_at, expires_at) VALUES (?, ?, ?, ?, 1, ?, ?)`,
		key.ID, key.Name, key.Prefix, secretHash(key.Raw), utcSeconds(time.Now()), nullableSeconds(expiresAt)); err != nil {
		return CreatedKey{}, fmt.Errorf("auth: create api key: %w", err)
	}
	return key, nil
}

// RotateKey disables the current key and creates its replacement with the same
// name and expiry in one transaction. A missing or already disabled id returns
// ErrKeyNotFound and creates nothing.
func (s *Service) RotateKey(ctx context.Context, id string) (CreatedKey, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CreatedKey{}, fmt.Errorf("auth: rotate api key: %w", err)
	}
	defer tx.Rollback()

	var (
		name      string
		expiresAt sql.NullString
	)
	err = tx.QueryRowContext(ctx,
		`SELECT name, expires_at FROM api_keys WHERE id = ? AND active = 1`, id).Scan(&name, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return CreatedKey{}, ErrKeyNotFound
	}
	if err != nil {
		return CreatedKey{}, fmt.Errorf("auth: rotate api key: %w", err)
	}

	var expiry *time.Time
	if expiresAt.Valid {
		parsed, err := time.Parse(time.RFC3339, expiresAt.String)
		if err != nil {
			return CreatedKey{}, fmt.Errorf("auth: api key %s has malformed expires_at %q: %w", id, expiresAt.String, err)
		}
		expiry = &parsed
	}
	replacement, err := newKey(name, expiry)
	if err != nil {
		return CreatedKey{}, err
	}

	now := utcSeconds(time.Now())
	res, err := tx.ExecContext(ctx,
		`UPDATE api_keys SET active = 0, disabled_at = ? WHERE id = ? AND active = 1`, now, id)
	if err != nil {
		return CreatedKey{}, fmt.Errorf("auth: rotate api key: %w", err)
	}
	disabled, err := res.RowsAffected()
	if err != nil {
		return CreatedKey{}, fmt.Errorf("auth: rotate api key: %w", err)
	}
	if disabled == 0 {
		return CreatedKey{}, ErrKeyNotFound
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO api_keys (id, name, prefix, key_hash, active, created_at, expires_at) VALUES (?, ?, ?, ?, 1, ?, ?)`,
		replacement.ID, replacement.Name, replacement.Prefix, secretHash(replacement.Raw), now, nullableSeconds(expiry)); err != nil {
		return CreatedKey{}, fmt.Errorf("auth: rotate api key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return CreatedKey{}, fmt.Errorf("auth: rotate api key: %w", err)
	}
	return replacement, nil
}

// DisableKey revokes a key. Repeating it keeps the first disabled_at value.
func (s *Service) DisableKey(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET active = 0, disabled_at = COALESCE(disabled_at, ?) WHERE id = ?`,
		utcSeconds(time.Now()), id)
	if err != nil {
		return fmt.Errorf("auth: disable api key: %w", err)
	}
	disabled, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: disable api key: %w", err)
	}
	if disabled == 0 {
		return ErrKeyNotFound
	}
	return nil
}

// ValidateKey reports whether raw is an active, unexpired client key at now.
// Keys grant read access only and are deliberately separate from sessions.
func (s *Service) ValidateKey(ctx context.Context, raw string, now time.Time) error {
	var (
		active    int
		expiresAt sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT active, expires_at FROM api_keys WHERE key_hash = ?`, secretHash(raw)).Scan(&active, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrKeyNotFound
	}
	if err != nil {
		return fmt.Errorf("auth: validate api key: %w", err)
	}
	if active == 0 {
		return ErrKeyNotFound
	}
	if !expiresAt.Valid {
		return nil
	}
	expires, err := time.Parse(time.RFC3339, expiresAt.String)
	if err != nil {
		return fmt.Errorf("auth: api key has malformed expires_at %q: %w", expiresAt.String, err)
	}
	if !now.Before(expires) {
		return ErrKeyNotFound
	}
	return nil
}

// Key is the metadata of one client key. Neither the hash nor the raw value is
// readable after creation.
type Key struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Active     bool       `json:"active"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	DisabledAt *time.Time `json:"disabled_at"`
}

// ListKeys returns every key newest first; only metadata, never key_hash.
func (s *Service) ListKeys(ctx context.Context) ([]Key, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, prefix, active, created_at, expires_at, disabled_at
		FROM api_keys ORDER BY created_at DESC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("auth: list api keys: %w", err)
	}
	defer rows.Close()

	items := []Key{}
	for rows.Next() {
		var (
			key                   Key
			active                int
			createdAt             string
			expiresAt, disabledAt sql.NullString
		)
		if err := rows.Scan(&key.ID, &key.Name, &key.Prefix, &active, &createdAt, &expiresAt, &disabledAt); err != nil {
			return nil, fmt.Errorf("auth: scan api key: %w", err)
		}
		key.Active = active == 1
		key.CreatedAt = parseSeconds(createdAt)
		key.ExpiresAt = optionalSeconds(expiresAt)
		key.DisabledAt = optionalSeconds(disabledAt)
		items = append(items, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth: read api keys: %w", err)
	}
	return items, nil
}

// parseSeconds reads the canonical UTC-second text timestamp.
func parseSeconds(value string) time.Time {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func optionalSeconds(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed := parseSeconds(value.String)
	return &parsed
}

func newKey(name string, expiresAt *time.Time) (CreatedKey, error) {
	secret, err := randomSecret()
	if err != nil {
		return CreatedKey{}, err
	}
	raw := apiKeyPrefix + secret
	return CreatedKey{ID: uuid.NewString(), Name: name, Prefix: raw[:len(apiKeyPrefix)+8], Raw: raw}, nil
}

func nullableSeconds(t *time.Time) any {
	if t == nil {
		return nil
	}
	return utcSeconds(*t)
}
