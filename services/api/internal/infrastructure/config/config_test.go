package config

import (
	"net/netip"
	"testing"
	"time"
)

func clearFactoryEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"RESTICK_DATA_DIR",
		"RESTICK_HTTP_ADDR",
		"RESTICK_PUBLIC_BASE_URL",
		"RESTICK_ADMIN_PASSWORD_FILE",
		"RESTICK_PROCESS_TIMEOUT",
		"RESTICK_TRUSTED_PROXY_CIDRS",
	} {
		t.Setenv(key, "")
	}
}

func TestConfigRejectsMissingPublicOrigin(t *testing.T) {
	clearFactoryEnv(t)
	if _, err := LoadFactory(); err == nil {
		t.Fatal("LoadFactory accepted a configuration without RESTICK_PUBLIC_BASE_URL")
	}
}

func TestConfigRejectsInvalidPublicOrigin(t *testing.T) {
	for _, raw := range []string{
		"http://restick.example.com",
		"restick.example.com",
		"https://",
		"https://user:pass@restick.example.com",
		"https://restick.example.com/api",
		"https://restick.example.com?x=1",
	} {
		clearFactoryEnv(t)
		t.Setenv("RESTICK_PUBLIC_BASE_URL", raw)
		if _, err := LoadFactory(); err == nil {
			t.Errorf("LoadFactory accepted public origin %q", raw)
		}
	}
}

func TestConfigFactoryDefaultsAndOverrides(t *testing.T) {
	clearFactoryEnv(t)
	t.Setenv("RESTICK_PUBLIC_BASE_URL", "https://restick.example.com/")
	cfg, err := LoadFactory()
	if err != nil {
		t.Fatalf("LoadFactory: %v", err)
	}
	if cfg.PublicBaseURL != "https://restick.example.com" {
		t.Errorf("PublicBaseURL = %q", cfg.PublicBaseURL)
	}
	if cfg.DataDir != "/data" {
		t.Errorf("DataDir = %q", cfg.DataDir)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.ProcessTimeout != 300*time.Second {
		t.Errorf("ProcessTimeout = %v", cfg.ProcessTimeout)
	}
	if cfg.AdminPasswordFile != "" || len(cfg.TrustedProxyCIDRs) != 0 {
		t.Errorf("unset optional fields = %q/%v", cfg.AdminPasswordFile, cfg.TrustedProxyCIDRs)
	}

	clearFactoryEnv(t)
	t.Setenv("RESTICK_PUBLIC_BASE_URL", "https://restick.example.com:8443")
	t.Setenv("RESTICK_DATA_DIR", "/srv/restick")
	t.Setenv("RESTICK_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("RESTICK_ADMIN_PASSWORD_FILE", "/run/secrets/admin")
	t.Setenv("RESTICK_PROCESS_TIMEOUT", "45s")
	t.Setenv("RESTICK_TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 192.168.1.7/24")
	cfg, err = LoadFactory()
	if err != nil {
		t.Fatalf("LoadFactory overrides: %v", err)
	}
	if cfg.PublicBaseURL != "https://restick.example.com:8443" || cfg.DataDir != "/srv/restick" ||
		cfg.HTTPAddr != "127.0.0.1:9090" || cfg.AdminPasswordFile != "/run/secrets/admin" ||
		cfg.ProcessTimeout != 45*time.Second {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
	want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("192.168.1.0/24")}
	if len(cfg.TrustedProxyCIDRs) != len(want) || cfg.TrustedProxyCIDRs[0] != want[0] || cfg.TrustedProxyCIDRs[1] != want[1] {
		t.Fatalf("TrustedProxyCIDRs = %v, want %v", cfg.TrustedProxyCIDRs, want)
	}
}

func TestConfigRejectsInvalidProcessSettings(t *testing.T) {
	clearFactoryEnv(t)
	t.Setenv("RESTICK_PUBLIC_BASE_URL", "https://restick.example.com")

	for _, tc := range []struct{ key, value string }{
		{"RESTICK_PROCESS_TIMEOUT", "0s"},
		{"RESTICK_PROCESS_TIMEOUT", "soon"},
		{"RESTICK_TRUSTED_PROXY_CIDRS", "10.0.0.1"},
	} {
		clearFactoryEnv(t)
		t.Setenv("RESTICK_PUBLIC_BASE_URL", "https://restick.example.com")
		t.Setenv(tc.key, tc.value)
		if _, err := LoadFactory(); err == nil {
			t.Errorf("LoadFactory accepted %s=%q", tc.key, tc.value)
		}
	}
}
