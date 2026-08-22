package config

import (
	"os"
	"strings"
)

type Config struct {
	ServiceName     string
	HTTPAddr        string
	GRPCAddr        string
	APISvcGRPCAddr  string
	FTPSvcGRPCAddr  string
	ImgToolGRPCAddr string
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucket     string
	MinIOUseSSL     bool
	InternalToken   string
	JWTSecret       string
	APIKeys         []string
	WebOrigin       string
}

func Load() Config {
	return Config{
		ServiceName:     envOr("SERVICE_NAME", "ftp_svc"),
		HTTPAddr:        envOr("HTTP_ADDR", ":8080"),
		GRPCAddr:        envOr("GRPC_ADDR", ":50052"),
		APISvcGRPCAddr:  strings.TrimSpace(os.Getenv("API_SVC_GRPC_ADDR")),
		FTPSvcGRPCAddr:  strings.TrimSpace(os.Getenv("FTP_SVC_GRPC_ADDR")),
		ImgToolGRPCAddr: strings.TrimSpace(os.Getenv("IMG_TOOL_GRPC_ADDR")),
		MinIOEndpoint:   envOr("MINIO_ENDPOINT", "minio:9000"),
		MinIOAccessKey:  envOr("MINIO_ACCESS_KEY", "restick"),
		MinIOSecretKey:  envOr("MINIO_SECRET_KEY", "resticksecret"),
		MinIOBucket:     envOr("MINIO_BUCKET", "restick-assets"),
		MinIOUseSSL:     envOr("MINIO_USE_SSL", "false") == "true",
		InternalToken:   envOr("INTERNAL_TOKEN", ""),
		JWTSecret:       envOr("JWT_SECRET", "change-me-jwt"),
		APIKeys:         splitCSV(envOr("API_KEYS", envOr("BOOTSTRAP_API_KEY", "dev-api-key"))),
		WebOrigin:       envOr("WEB_ORIGIN", "http://localhost:8088"),
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
