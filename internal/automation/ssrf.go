package automation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// HTTPPolicy controls the outbound requests made by the Lua script runtime.
type HTTPPolicy struct {
	// AllowedHosts is an optional list of permitted hosts. Entries may be bare
	// hostnames ("example.com") or "host:port" pairs ("api.internal:8080").
	// Matching is case-insensitive. Hosts listed here bypass the IP-based
	// SSRF checks and are always reachable.
	AllowedHosts []string
}

// forbiddenPrefixes are the IPv4/IPv6 ranges that outbound script requests may
// never reach: loopback, link-local, private, and reserved/benchmark/test
// networks. Requests resolving to any of these are rejected.
var forbiddenPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // "this" network / unspecified
	netip.MustParsePrefix("10.0.0.0/8"),      // private
	netip.MustParsePrefix("100.64.0.0/10"),   // carrier-grade NAT
	netip.MustParsePrefix("127.0.0.0/8"),     // loopback
	netip.MustParsePrefix("169.254.0.0/16"),  // link-local
	netip.MustParsePrefix("172.16.0.0/12"),   // private
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1
	netip.MustParsePrefix("192.168.0.0/16"),  // private
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
	netip.MustParsePrefix("224.0.0.0/4"),     // multicast
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved
	netip.MustParsePrefix("::1/128"),         // loopback
	netip.MustParsePrefix("::/128"),          // unspecified
	netip.MustParsePrefix("100::/64"),        // discard-only
	netip.MustParsePrefix("2001:db8::/32"),   // documentation
	netip.MustParsePrefix("fc00::/7"),        // unique-local
	netip.MustParsePrefix("fe80::/10"),       // link-local
	netip.MustParsePrefix("ff00::/8"),        // multicast
}

// isForbiddenIP reports whether ip falls inside a range the script runtime is
// not allowed to contact. IPv4-mapped IPv6 addresses are unmapped first.
func isForbiddenIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return true
	}
	for _, p := range forbiddenPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// validateRequestURL rejects URLs the script runtime must not fetch: anything
// that is not http(s) or that has no host.
func validateRequestURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("url has no host")
	}
	return nil
}

// newSafeHTTPClient returns an http.Client whose dialer blocks requests to
// forbidden IP ranges and only permits http(s) hosts. IP validation happens at
// connection time against the actual resolved addresses, mitigating DNS
// rebinding.
func newSafeHTTPClient(policy HTTPPolicy) *http.Client {
	allow := make(map[string]struct{}, len(policy.AllowedHosts))
	for _, h := range policy.AllowedHosts {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			allow[h] = struct{}{}
		}
	}

	base := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("ssrf: invalid address %q: %w", addr, err)
		}
		if hostAllowed(allow, host, port) {
			return base.DialContext(ctx, network, addr)
		}

		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("ssrf: resolve %q: %w", host, err)
		}
		for _, ip := range ips {
			if isForbiddenIP(ip) {
				return nil, fmt.Errorf("ssrf: blocked request to %s (resolves to %s)", host, ip)
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("ssrf: no addresses resolved for %q", host)
		}
		// Dial the validated IP directly so a later DNS answer cannot rebind
		// the connection to a forbidden address.
		return base.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext:           dialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
}

// hostAllowed reports whether host (or host:port) is explicitly permitted.
func hostAllowed(allow map[string]struct{}, host, port string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if _, ok := allow[host]; ok {
		return true
	}
	if port != "" {
		if _, ok := allow[net.JoinHostPort(host, port)]; ok {
			return true
		}
	}
	return false
}
