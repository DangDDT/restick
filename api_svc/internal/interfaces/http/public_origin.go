package httpx

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func clientReachableOrigin(configured string, r *http.Request) string {
	configured = strings.TrimRight(configured, "/")
	parsed, err := url.Parse(configured)
	if err != nil || parsed.Host == "" {
		return configured
	}
	if !isLoopbackHost(parsed.Hostname()) {
		return configured
	}
	requestHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if requestHost == "" {
		requestHost = r.Host
	}
	if i := strings.Index(requestHost, ","); i >= 0 {
		requestHost = strings.TrimSpace(requestHost[:i])
	}
	clientHost, _, splitErr := net.SplitHostPort(requestHost)
	if splitErr != nil {
		clientHost = requestHost
	}
	clientHost = strings.Trim(clientHost, "[]")
	if clientHost == "" || isLoopbackHost(clientHost) {
		return configured
	}
	ftpPort := parsed.Port()
	if ftpPort == "" {
		ftpPort = "80"
		if parsed.Scheme == "https" {
			ftpPort = "443"
		}
	}
	parsed.Host = net.JoinHostPort(clientHost, ftpPort)
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto == "http" || proto == "https" {
		parsed.Scheme = proto
	}
	return strings.TrimRight(parsed.String(), "/")
}

func isLoopbackHost(host string) bool {
	normalized := strings.ToLower(strings.Trim(host, "[]"))
	return normalized == "localhost" || normalized == "127.0.0.1" || normalized == "::1" || normalized == "0.0.0.0"
}
