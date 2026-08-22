package apikey

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateRawKey(t *testing.T) {
	raw, prefix, hash, err := GenerateRawKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(raw, "rstk_live_") {
		t.Errorf("raw key does not have rstk_live_ prefix: %s", raw)
	}
	if !strings.HasPrefix(prefix, "rstk_live_") || !strings.Contains(prefix, "...") {
		t.Errorf("invalid prefix format: %s", prefix)
	}
	if hash != HashKey(raw) {
		t.Errorf("hash does not match HashKey(raw)")
	}
}

func TestIsExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	keyNoExpiry := APIKey{}
	if keyNoExpiry.IsExpired() {
		t.Errorf("expected key with no expiry to not be expired")
	}

	keyExpired := APIKey{ExpiresAt: &past}
	if !keyExpired.IsExpired() {
		t.Errorf("expected key with past expiry to be expired")
	}

	keyActive := APIKey{ExpiresAt: &future}
	if keyActive.IsExpired() {
		t.Errorf("expected key with future expiry to not be expired")
	}
}

func TestGenerateRawKeyUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		raw, _, hash, err := GenerateRawKey()
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if seen[raw] || seen[hash] {
			t.Fatalf("duplicate key generated at iteration %d", i)
		}
		seen[raw] = true
		seen[hash] = true
	}
}
