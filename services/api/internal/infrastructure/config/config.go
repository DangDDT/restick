package config

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
)

// FactoryConfig is the validated configuration of the single-operator factory
// runtime (spec §12). PublicBaseURL is always an HTTPS origin supplied by the
// operator, never derived from a request Host header.
type FactoryConfig struct {
	DataDir           string
	HTTPAddr          string
	PublicBaseURL     string
	AdminPasswordFile string
	ProcessTimeout    time.Duration
	TrustedProxyCIDRs []netip.Prefix
}

func LoadFactory() (FactoryConfig, error) {
	cfg := FactoryConfig{
		DataDir:           envOr("RESTICK_DATA_DIR", "/data"),
		HTTPAddr:          envOr("RESTICK_HTTP_ADDR", ":8080"),
		AdminPasswordFile: strings.TrimSpace(os.Getenv("RESTICK_ADMIN_PASSWORD_FILE")),
		ProcessTimeout:    300 * time.Second,
	}

	base, err := normalizePublicBaseURL(strings.TrimSpace(os.Getenv("RESTICK_PUBLIC_BASE_URL")))
	if err != nil {
		return FactoryConfig{}, err
	}
	cfg.PublicBaseURL = base

	if raw := strings.TrimSpace(os.Getenv("RESTICK_PROCESS_TIMEOUT")); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return FactoryConfig{}, fmt.Errorf("config: RESTICK_PROCESS_TIMEOUT must be a positive duration, got %q", raw)
		}
		cfg.ProcessTimeout = timeout
	}

	cidrs, err := parseCIDRs(os.Getenv("RESTICK_TRUSTED_PROXY_CIDRS"))
	if err != nil {
		return FactoryConfig{}, err
	}
	cfg.TrustedProxyCIDRs = cidrs
	return cfg, nil
}

func normalizePublicBaseURL(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("config: RESTICK_PUBLIC_BASE_URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("config: RESTICK_PUBLIC_BASE_URL is not a valid URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("config: RESTICK_PUBLIC_BASE_URL must be an https origin, got %q", raw)
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("config: RESTICK_PUBLIC_BASE_URL must not carry a path, query or fragment, got %q", raw)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func parseCIDRs(raw string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			return nil, fmt.Errorf("config: RESTICK_TRUSTED_PROXY_CIDRS entry %q is not a CIDR prefix: %w", part, err)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	if len(prefixes) == 0 {
		return []netip.Prefix{}, nil
	}
	return prefixes, nil
}

type Config struct {
	ServiceName         string
	HTTPAddr            string
	GRPCAddr            string
	APISvcGRPCAddr      string
	FTPSvcGRPCAddr      string
	ImgToolGRPCAddr     string
	PostgresDSN         string
	JWTSecret           string
	InternalToken       string
	WebOrigins          []string
	PublicFTPOrigin     string
	BootstrapAdminEmail string
	BootstrapAdminPass  string
	BootstrapAPIKey     string
}

func Load() Config {
	return Config{
		ServiceName:         envOr("SERVICE_NAME", "api_svc"),
		HTTPAddr:            envOr("HTTP_ADDR", ":8080"),
		GRPCAddr:            envOr("GRPC_ADDR", ":50051"),
		APISvcGRPCAddr:      strings.TrimSpace(os.Getenv("API_SVC_GRPC_ADDR")),
		FTPSvcGRPCAddr:      strings.TrimSpace(os.Getenv("FTP_SVC_GRPC_ADDR")),
		ImgToolGRPCAddr:     strings.TrimSpace(os.Getenv("IMG_TOOL_GRPC_ADDR")),
		PostgresDSN:         envOr("POSTGRES_DSN", "postgres://restick:restick@postgres:5432/restick?sslmode=disable"),
		JWTSecret:           envOr("JWT_SECRET", "change-me-jwt"),
		InternalToken:       envOr("INTERNAL_TOKEN", ""),
		WebOrigins:          splitCSV(envOr("WEB_ORIGIN", "http://localhost")),
		PublicFTPOrigin:     envOr("PUBLIC_FTP_ORIGIN", "http://localhost"),
		BootstrapAdminEmail: envOr("BOOTSTRAP_ADMIN_EMAIL", "admin@restick.local"),
		BootstrapAdminPass:  envOr("BOOTSTRAP_ADMIN_PASSWORD", "changeme"),
		BootstrapAPIKey:     envOr("BOOTSTRAP_API_KEY", "dev-api-key"),
	}
}

func envOr(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
