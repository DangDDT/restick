package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Login verifies the owner password and returns the raw session token. Only the
// token's SHA256 digest is persisted; the token itself is never recoverable.
func (s *Service) Login(ctx context.Context, password string, now time.Time) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM owner WHERE id = 1`).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNoOwner
	}
	if err != nil {
		return "", fmt.Errorf("auth: read owner: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}

	raw, err := randomSecret()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO admin_sessions (id, token_hash, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), secretHash(raw), utcSeconds(now), utcSeconds(now.Add(SessionTTL)))
	if err != nil {
		return "", fmt.Errorf("auth: create session: %w", err)
	}
	return raw, nil
}

// ValidateSession reports whether raw is a live admin session at now. The
// expiry instant itself is already expired.
func (s *Service) ValidateSession(ctx context.Context, raw string, now time.Time) error {
	var expiresAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT expires_at FROM admin_sessions WHERE token_hash = ?`, secretHash(raw)).Scan(&expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSessionInvalid
	}
	if err != nil {
		return fmt.Errorf("auth: validate session: %w", err)
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return fmt.Errorf("auth: session has malformed expires_at %q: %w", expiresAt, err)
	}
	if !now.Before(expires) {
		return ErrSessionInvalid
	}
	return nil
}

// Logout deletes the session server-side. An unknown token is a no-op.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token_hash = ?`, secretHash(raw)); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	return nil
}
