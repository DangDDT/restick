package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound  = errors.New("api key not found")
	ErrForbidden = errors.New("forbidden")
)

type APIKey struct {
	ID         string     `json:"id"`
	UserID     *string    `json:"user_id,omitempty"`
	UserEmail  string     `json:"user_email,omitempty"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	IsActive   bool       `json:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreatedAPIKey struct {
	APIKey
	RawKey string `json:"raw_key"`
}

func GenerateRawKey() (string, string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", "", err
	}
	randomHex := hex.EncodeToString(buf)
	rawKey := fmt.Sprintf("rstk_live_%s", randomHex)
	prefix := fmt.Sprintf("rstk_live_%s...%s", randomHex[:6], randomHex[len(randomHex)-4:])
	hash := HashKey(rawKey)
	return rawKey, prefix, hash, nil
}

func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (k APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}
