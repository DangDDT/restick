package config

import (
	"os"
	"strings"
)

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
	WebOrigin           string
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
		WebOrigin:           envOr("WEB_ORIGIN", "http://localhost:8088"),
		PublicFTPOrigin:     envOr("PUBLIC_FTP_ORIGIN", "http://localhost:8081"),
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
