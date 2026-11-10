package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/storage"
)

const ownerPassword = "correct horse battery staple"

func openTemp(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(db), db
}

func mustBootstrap(t *testing.T, svc *Service) {
	t.Helper()
	if err := svc.BootstrapOwner(context.Background(), []byte(ownerPassword)); err != nil {
		t.Fatalf("BootstrapOwner: %v", err)
	}
}

func mustLogin(t *testing.T, svc *Service, now time.Time) string {
	t.Helper()
	raw, err := svc.Login(context.Background(), ownerPassword, now)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if raw == "" {
		t.Fatal("Login returned an empty session token")
	}
	return raw
}

func TestOwnerBootstrapDoesNotOverwritePassword(t *testing.T) {
	ctx := context.Background()
	svc, db := openTemp(t)

	if err := svc.BootstrapOwner(ctx, []byte(ownerPassword)); err != nil {
		t.Fatalf("first BootstrapOwner: %v", err)
	}
	if err := svc.BootstrapOwner(ctx, []byte("replacement password")); !errors.Is(err, ErrOwnerExists) {
		t.Fatalf("second BootstrapOwner = %v, want ErrOwnerExists", err)
	}
	if err := svc.BootstrapOwner(ctx, nil); !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("empty BootstrapOwner = %v, want ErrEmptyPassword", err)
	}

	var owners int
	if err := db.QueryRow(`SELECT count(*) FROM owner`).Scan(&owners); err != nil {
		t.Fatalf("count owners: %v", err)
	}
	if owners != 1 {
		t.Fatalf("owner rows = %d, want 1", owners)
	}

	if _, err := svc.Login(ctx, ownerPassword, time.Now()); err != nil {
		t.Fatalf("Login with original password: %v", err)
	}
	if _, err := svc.Login(ctx, "replacement password", time.Now()); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login with bootstrap-replacement password = %v, want ErrInvalidCredentials", err)
	}
}

func TestSessionExpiryAndLogout(t *testing.T) {
	ctx := context.Background()
	svc, db := openTemp(t)
	mustBootstrap(t, svc)

	now := time.Date(2026, 11, 10, 9, 30, 15, 0, time.UTC)
	raw := mustLogin(t, svc, now)

	if err := svc.ValidateSession(ctx, raw, now.Add(23*time.Hour)); err != nil {
		t.Fatalf("ValidateSession within TTL: %v", err)
	}
	if err := svc.ValidateSession(ctx, raw, now.Add(SessionTTL+time.Second)); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession past TTL = %v, want ErrSessionInvalid", err)
	}
	if err := svc.ValidateSession(ctx, "not-a-session", now); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession unknown token = %v, want ErrSessionInvalid", err)
	}

	var storedExpiry string
	if err := db.QueryRow(`SELECT expires_at FROM admin_sessions WHERE token_hash = ?`, secretHash(raw)).Scan(&storedExpiry); err != nil {
		t.Fatalf("read session expiry: %v", err)
	}
	if want := "2026-11-11T09:30:15Z"; storedExpiry != want {
		t.Fatalf("expires_at = %q, want %q", storedExpiry, want)
	}

	if err := svc.Logout(ctx, raw); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if err := svc.ValidateSession(ctx, raw, now.Add(time.Minute)); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession after logout = %v, want ErrSessionInvalid", err)
	}
	if err := svc.Logout(ctx, raw); err != nil {
		t.Fatalf("second Logout = %v, want nil", err)
	}
}

func TestPasswordResetRevokesSessions(t *testing.T) {
	ctx := context.Background()
	svc, _ := openTemp(t)
	mustBootstrap(t, svc)

	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	first := mustLogin(t, svc, now)
	second := mustLogin(t, svc, now.Add(time.Minute))

	if err := svc.SetPassword(ctx, []byte("rotated password")); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	for i, raw := range []string{first, second} {
		if err := svc.ValidateSession(ctx, raw, now.Add(2*time.Minute)); !errors.Is(err, ErrSessionInvalid) {
			t.Fatalf("session %d after reset = %v, want ErrSessionInvalid", i, err)
		}
	}
	if _, err := svc.Login(ctx, ownerPassword, now); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login with old password = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Login(ctx, "rotated password", now); err != nil {
		t.Fatalf("Login with new password: %v", err)
	}

	emptySvc, _ := openTemp(t)
	if err := emptySvc.SetPassword(ctx, []byte("nobody")); !errors.Is(err, ErrNoOwner) {
		t.Fatalf("SetPassword without owner = %v, want ErrNoOwner", err)
	}
}

func TestRotateDisablesOldKeyAtomically(t *testing.T) {
	ctx := context.Background()
	svc, db := openTemp(t)
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	expires := now.Add(30 * 24 * time.Hour)
	old, err := svc.CreateKey(ctx, "phone", &expires)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if !strings.HasPrefix(old.Raw, old.Prefix) {
		t.Fatalf("prefix %q is not a prefix of raw key", old.Prefix)
	}
	if err := svc.ValidateKey(ctx, old.Raw, now); err != nil {
		t.Fatalf("ValidateKey new key: %v", err)
	}

	replacement, err := svc.RotateKey(ctx, old.ID)
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	if replacement.ID == old.ID || replacement.Raw == old.Raw {
		t.Fatalf("rotation reused identity: %+v", replacement)
	}
	if replacement.Name != "phone" {
		t.Fatalf("rotated name = %q, want phone", replacement.Name)
	}
	if err := svc.ValidateKey(ctx, old.Raw, now); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("ValidateKey old key = %v, want ErrKeyNotFound", err)
	}
	if err := svc.ValidateKey(ctx, replacement.Raw, now); err != nil {
		t.Fatalf("ValidateKey replacement key: %v", err)
	}

	var (
		active     int
		disabledAt sql.NullString
		newExpiry  sql.NullString
	)
	if err := db.QueryRow(`SELECT active, disabled_at FROM api_keys WHERE id = ?`, old.ID).Scan(&active, &disabledAt); err != nil {
		t.Fatalf("read old key: %v", err)
	}
	if active != 0 || !disabledAt.Valid {
		t.Fatalf("old key active=%d disabled_at=%v, want disabled", active, disabledAt)
	}
	if err := db.QueryRow(`SELECT expires_at FROM api_keys WHERE id = ?`, replacement.ID).Scan(&newExpiry); err != nil {
		t.Fatalf("read replacement key: %v", err)
	}
	if want := utcSeconds(expires); newExpiry.String != want {
		t.Fatalf("replacement expires_at = %q, want %q", newExpiry.String, want)
	}

	var before int
	if err := db.QueryRow(`SELECT count(*) FROM api_keys`).Scan(&before); err != nil {
		t.Fatalf("count api_keys: %v", err)
	}
	if _, err := svc.RotateKey(ctx, "missing-key"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("RotateKey(unknown) = %v, want ErrKeyNotFound", err)
	}
	if _, err := svc.RotateKey(ctx, old.ID); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("RotateKey(disabled) = %v, want ErrKeyNotFound", err)
	}
	var after int
	if err := db.QueryRow(`SELECT count(*) FROM api_keys`).Scan(&after); err != nil {
		t.Fatalf("count api_keys: %v", err)
	}
	if after != before {
		t.Fatalf("failed rotations changed key count %d -> %d", before, after)
	}

	if err := svc.DisableKey(ctx, replacement.ID); err != nil {
		t.Fatalf("DisableKey: %v", err)
	}
	if err := svc.DisableKey(ctx, replacement.ID); err != nil {
		t.Fatalf("second DisableKey = %v, want nil", err)
	}
	if err := svc.ValidateKey(ctx, replacement.Raw, now); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("ValidateKey disabled key = %v, want ErrKeyNotFound", err)
	}
	if err := svc.DisableKey(ctx, "missing-key"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("DisableKey(unknown) = %v, want ErrKeyNotFound", err)
	}
}

func TestOnlyHashesPersisted(t *testing.T) {
	ctx := context.Background()
	svc, db := openTemp(t)
	mustBootstrap(t, svc)

	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	session := mustLogin(t, svc, now)
	key, err := svc.CreateKey(ctx, "laptop", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	var passwordHash string
	if err := db.QueryRow(`SELECT password_hash FROM owner`).Scan(&passwordHash); err != nil {
		t.Fatalf("read owner hash: %v", err)
	}
	if passwordHash == ownerPassword || strings.Contains(passwordHash, ownerPassword) {
		t.Fatalf("owner password stored in plaintext: %q", passwordHash)
	}
	if !strings.HasPrefix(passwordHash, "$2") {
		t.Fatalf("owner password_hash = %q, want bcrypt", passwordHash)
	}

	var tokenHash []byte
	if err := db.QueryRow(`SELECT token_hash FROM admin_sessions`).Scan(&tokenHash); err != nil {
		t.Fatalf("read session hash: %v", err)
	}
	want := sha256.Sum256([]byte(session))
	if !bytes.Equal(tokenHash, want[:]) {
		t.Fatalf("token_hash is not sha256(raw session)")
	}
	if bytes.Equal(tokenHash, []byte(session)) {
		t.Fatal("raw session token stored as token_hash")
	}
	var leakedSessions int
	if err := db.QueryRow(`SELECT count(*) FROM admin_sessions WHERE id = ?`, session).Scan(&leakedSessions); err != nil {
		t.Fatalf("count leaked sessions: %v", err)
	}
	if leakedSessions != 0 {
		t.Fatal("raw session token stored in admin_sessions.id")
	}

	var (
		keyHash []byte
		name    string
		prefix  string
	)
	if err := db.QueryRow(`SELECT key_hash, name, prefix FROM api_keys WHERE id = ?`, key.ID).Scan(&keyHash, &name, &prefix); err != nil {
		t.Fatalf("read api key: %v", err)
	}
	wantKeyHash := sha256.Sum256([]byte(key.Raw))
	if !bytes.Equal(keyHash, wantKeyHash[:]) {
		t.Fatalf("key_hash is not sha256(raw key)")
	}
	if name != "laptop" || !strings.HasPrefix(key.Raw, prefix) || prefix == key.Raw {
		t.Fatalf("stored key metadata name=%q prefix=%q raw=%q", name, prefix, key.Raw)
	}
	var leakedKeys int
	if err := db.QueryRow(`SELECT count(*) FROM api_keys WHERE id = ? OR name = ? OR prefix = ?`, key.Raw, key.Raw, key.Raw).Scan(&leakedKeys); err != nil {
		t.Fatalf("count leaked keys: %v", err)
	}
	if leakedKeys != 0 {
		t.Fatal("raw api key stored in api_keys columns")
	}
}

func TestExpiryBoundary(t *testing.T) {
	ctx := context.Background()
	svc, db := openTemp(t)
	mustBootstrap(t, svc)

	now := time.Date(2026, 11, 10, 9, 0, 0, 500_000_000, time.UTC)
	session := mustLogin(t, svc, now)

	var sessionExpiry string
	if err := db.QueryRow(`SELECT expires_at FROM admin_sessions WHERE token_hash = ?`, secretHash(session)).Scan(&sessionExpiry); err != nil {
		t.Fatalf("read session expiry: %v", err)
	}
	expires, err := time.Parse(time.RFC3339, sessionExpiry)
	if err != nil {
		t.Fatalf("parse session expiry %q: %v", sessionExpiry, err)
	}
	if got := expires.Sub(now); got != SessionTTL-500*time.Millisecond {
		t.Fatalf("session lifetime = %v, want TTL truncated to the second", got)
	}
	if err := svc.ValidateSession(ctx, session, expires.Add(-time.Second)); err != nil {
		t.Fatalf("ValidateSession one second before expiry: %v", err)
	}
	if err := svc.ValidateSession(ctx, session, expires); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession at expiry = %v, want ErrSessionInvalid", err)
	}
	if err := svc.ValidateSession(ctx, session, expires.Add(-time.Nanosecond)); err != nil {
		t.Fatalf("ValidateSession a nanosecond before expiry: %v", err)
	}
	if err := svc.ValidateSession(ctx, session, expires.Add(time.Nanosecond)); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("ValidateSession a nanosecond after expiry = %v, want ErrSessionInvalid", err)
	}

	keyExpiry := now.Add(2 * time.Hour)
	key, err := svc.CreateKey(ctx, "boundary", &keyExpiry)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	var keyExpiryStored string
	if err := db.QueryRow(`SELECT expires_at FROM api_keys WHERE id = ?`, key.ID).Scan(&keyExpiryStored); err != nil {
		t.Fatalf("read key expiry: %v", err)
	}
	keyExpires, err := time.Parse(time.RFC3339, keyExpiryStored)
	if err != nil {
		t.Fatalf("parse key expiry %q: %v", keyExpiryStored, err)
	}
	if err := svc.ValidateKey(ctx, key.Raw, keyExpires.Add(-time.Second)); err != nil {
		t.Fatalf("ValidateKey one second before expiry: %v", err)
	}
	if err := svc.ValidateKey(ctx, key.Raw, keyExpires); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("ValidateKey at expiry = %v, want ErrKeyNotFound", err)
	}
}

func TestForwardedIPRequiresTrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("2001:db8:1::/48"),
	}
	cases := []struct {
		name string
		peer string
		xff  string
		want netip.Addr
	}{
		{
			name: "peer outside trusted CIDRs ignores chain",
			peer: "198.51.100.7:41234",
			xff:  "203.0.113.9, 10.0.0.4",
			want: netip.MustParseAddr("198.51.100.7"),
		},
		{
			name: "trusted peer skips trusted hops right to left",
			peer: "10.0.0.5:8443",
			xff:  "172.16.0.1, 203.0.113.9, 10.0.0.4",
			want: netip.MustParseAddr("203.0.113.9"),
		},
		{
			name: "forged left entry never wins",
			peer: "10.0.0.5",
			xff:  "1.2.3.4, 203.0.113.9",
			want: netip.MustParseAddr("203.0.113.9"),
		},
		{
			name: "malformed rightmost hop fails safe to peer",
			peer: "10.0.0.5",
			xff:  "203.0.113.9, not-an-ip",
			want: netip.MustParseAddr("10.0.0.5"),
		},
		{
			name: "empty chain keeps peer",
			peer: "10.0.0.5",
			xff:  "",
			want: netip.MustParseAddr("10.0.0.5"),
		},
		{
			name: "all hops trusted keeps leftmost",
			peer: "10.0.0.5",
			xff:  "10.0.0.6, 10.0.0.7",
			want: netip.MustParseAddr("10.0.0.6"),
		},
		{
			name: "ipv6 trusted peer",
			peer: "[2001:db8:1::5]:9000",
			xff:  "2001:db8:2::9",
			want: netip.MustParseAddr("2001:db8:2::9"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ClientSource(tc.peer, tc.xff, trusted)
			if err != nil {
				t.Fatalf("ClientSource(%q, %q): %v", tc.peer, tc.xff, err)
			}
			if got != tc.want {
				t.Fatalf("ClientSource(%q, %q) = %v, want %v", tc.peer, tc.xff, got, tc.want)
			}
		})
	}

	for _, peer := range []string{"", "unknown", "not an address"} {
		if _, err := ClientSource(peer, "203.0.113.9", trusted); !errors.Is(err, ErrInvalidPeer) {
			t.Fatalf("ClientSource peer %q = %v, want ErrInvalidPeer", peer, err)
		}
	}

	prefixes, err := ParseTrustedCIDRs([]string{"10.0.0.0/8", " 192.168.1.5 ", ""})
	if err != nil {
		t.Fatalf("ParseTrustedCIDRs: %v", err)
	}
	if len(prefixes) != 2 || !prefixes[1].Contains(netip.MustParseAddr("192.168.1.5")) {
		t.Fatalf("ParseTrustedCIDRs = %v, want 10.0.0.0/8 and host prefix", prefixes)
	}
	if _, err := ParseTrustedCIDRs([]string{"10.0.0.0/33"}); err == nil {
		t.Fatal("ParseTrustedCIDRs accepted an invalid CIDR")
	}
}
