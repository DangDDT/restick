package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientReachableOrigin(t *testing.T) {
	tests := []struct {
		name        string
		configured  string
		reqHost     string
		fwdHost     string
		fwdProto    string
		expected    string
	}{
		{
			name:       "localhost configured, reverse proxy with domain and https",
			configured: "http://localhost",
			reqHost:    "internal-ip:8080",
			fwdHost:    "restick.example.com",
			fwdProto:   "https",
			expected:   "https://restick.example.com",
		},
		{
			name:       "localhost configured, reverse proxy with domain and http",
			configured: "http://localhost",
			reqHost:    "internal-ip:8080",
			fwdHost:    "restick.example.com",
			fwdProto:   "http",
			expected:   "http://restick.example.com",
		},
		{
			name:       "localhost configured, direct mobile access with LAN IP in Host header",
			configured: "http://localhost",
			reqHost:    "192.168.1.50:8080",
			expected:   "http://192.168.1.50",
		},
		{
			name:       "non-loopback configured, stays untouched",
			configured: "https://cdn.example.com",
			reqHost:    "internal-ip:8080",
			fwdHost:    "restick.example.com",
			fwdProto:   "https",
			expected:   "https://cdn.example.com",
		},
		{
			name:       "loopback client host stays configured",
			configured: "http://localhost",
			reqHost:    "localhost:8080",
			expected:   "http://localhost",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
			if tc.reqHost != "" {
				req.Host = tc.reqHost
			}
			if tc.fwdHost != "" {
				req.Header.Set("X-Forwarded-Host", tc.fwdHost)
			}
			if tc.fwdProto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.fwdProto)
			}

			result := clientReachableOrigin(tc.configured, req)
			if result != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
