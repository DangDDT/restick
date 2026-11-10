// Package auth owns the single factory owner's password, the admin sessions and
// the read-only client API keys. Credentials are bcrypt-hashed; session tokens
// and API keys are stored as SHA256 digests only.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// OwnerUsername is the only owner account; the factory has exactly one.
const OwnerUsername = "admin"

// SessionTTL is the lifetime of an admin session.
const SessionTTL = 24 * time.Hour

// secretBytes is the raw size of a session token or API key.
const secretBytes = 32

var (
	ErrOwnerExists        = errors.New("auth: owner already bootstrapped")
	ErrNoOwner            = errors.New("auth: owner not bootstrapped")
	ErrEmptyPassword      = errors.New("auth: password must not be empty")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrSessionInvalid     = errors.New("auth: session invalid or expired")
	ErrKeyNotFound        = errors.New("auth: api key not found")
	ErrEmptyKeyName       = errors.New("auth: api key name must not be empty")
	ErrInvalidPeer        = errors.New("auth: invalid peer address")
)

// Service exposes owner, session, key and request-source operations over the
// factory SQLite database.
type Service struct {
	db *sql.DB
}

// NewService wraps db. The caller owns the connection (one connection by
// storage.Open) and its lifetime.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// BootstrapOwner creates the owner row. An already bootstrapped owner is left
// untouched and reported as ErrOwnerExists, so startup never resets a password.
func (s *Service) BootstrapOwner(ctx context.Context, password []byte) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO owner (id, username, password_hash) VALUES (1, ?, ?) ON CONFLICT (id) DO NOTHING`,
		OwnerUsername, hash)
	if err != nil {
		return fmt.Errorf("auth: create owner: %w", err)
	}
	created, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: create owner: %w", err)
	}
	if created == 0 {
		return ErrOwnerExists
	}
	return nil
}

// SetPassword replaces the owner password and revokes every admin session in
// one transaction.
func (s *Service) SetPassword(ctx context.Context, password []byte) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("auth: reset password: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE owner SET password_hash = ?, updated_at = ? WHERE id = 1`,
		hash, utcSeconds(time.Now()))
	if err != nil {
		return fmt.Errorf("auth: reset password: %w", err)
	}
	updated, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: reset password: %w", err)
	}
	if updated == 0 {
		return ErrNoOwner
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_sessions`); err != nil {
		return fmt.Errorf("auth: revoke sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("auth: reset password: %w", err)
	}
	return nil
}

func hashPassword(password []byte) (string, error) {
	if len(password) == 0 {
		return "", ErrEmptyPassword
	}
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// randomSecret returns 32 cryptographically random bytes as hex.
func randomSecret() (string, error) {
	buf := make([]byte, secretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: random secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// secretHash is the value persisted for a raw session token or API key.
func secretHash(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// utcSeconds renders the storage timestamp format: UTC, second precision.
func utcSeconds(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
