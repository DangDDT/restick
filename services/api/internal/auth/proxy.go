package auth

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// ParseTrustedCIDRs parses RESTICK_TRUSTED_PROXY_CIDRS values. A bare IP is
// accepted as a single-address prefix; empty entries are skipped.
func ParseTrustedCIDRs(specs []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(spec); err == nil {
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(spec)
		if err != nil {
			return nil, fmt.Errorf("auth: invalid trusted proxy %q: %w", spec, err)
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes, nil
}

// ClientSource derives the request source address for login throttling and
// logs. peerAddr is the immediate peer ("host:port", "[v6]:port" or bare host)
// and forwardedFor is the raw X-Forwarded-For header. Forwarded values are read
// right to left only while every hop is a trusted proxy; the first untrusted hop
// is the client, so a forged left-hand entry can never win. An untrusted peer
// ignores forwardedFor completely, and a malformed peer or hop ends the walk at
// the last address already proven, never at attacker-supplied text.
func ClientSource(peerAddr, forwardedFor string, trusted []netip.Prefix) (netip.Addr, error) {
	peer, err := parseAddr(peerAddr)
	if err != nil {
		return netip.Addr{}, ErrInvalidPeer
	}
	if !trustedAddr(peer, trusted) {
		return peer, nil
	}

	source := peer
	hops := strings.Split(forwardedFor, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, err := parseAddr(hops[i])
		if err != nil {
			return source, nil
		}
		source = hop
		if !trustedAddr(hop, trusted) {
			return hop, nil
		}
	}
	return source, nil
}

func parseAddr(raw string) (netip.Addr, error) {
	raw = strings.TrimSpace(raw)
	if addr, err := netip.ParseAddr(raw); err == nil {
		return addr.Unmap(), nil
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			return addr.Unmap(), nil
		}
	}
	return netip.Addr{}, ErrInvalidPeer
}

func trustedAddr(addr netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
