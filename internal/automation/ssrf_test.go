package automation

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestIsForbiddenIP(t *testing.T) {
	cases := []struct {
		ip     string
		forbid bool
	}{
		{"127.0.0.1", true},
		{"127.8.8.8", true},
		{"10.1.2.3", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.0.1", true},
		{"169.254.169.254", true},
		{"100.64.0.1", true},
		{"0.0.0.0", true},
		{"224.0.0.1", true},
		{"240.0.0.1", true},
		{"192.0.2.1", true},
		{"198.18.0.1", true},
		{"::1", true},
		{"::", true},
		{"fe80::1", true},
		{"fd00::1", true},
		{"fc00::1", true},
		{"ff02::1", true},
		{"::ffff:127.0.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false},
		{"2606:2800:220:1:248:1893:25c8:1946", false},
	}
	for _, c := range cases {
		ip := netip.MustParseAddr(c.ip)
		if got := isForbiddenIP(ip); got != c.forbid {
			t.Errorf("isForbiddenIP(%s) = %v, want %v", c.ip, got, c.forbid)
		}
	}
}

func TestValidateRequestURL(t *testing.T) {
	if err := validateRequestURL("https://example.com/x"); err != nil {
		t.Errorf("https url should be allowed: %v", err)
	}
	if err := validateRequestURL("http://example.com/x"); err != nil {
		t.Errorf("http url should be allowed: %v", err)
	}
	if err := validateRequestURL("file:///etc/passwd"); err == nil {
		t.Errorf("file scheme should be rejected")
	}
	if err := validateRequestURL("ftp://example.com"); err == nil {
		t.Errorf("ftp scheme should be rejected")
	}
	if err := validateRequestURL("gopher://example.com"); err == nil {
		t.Errorf("gopher scheme should be rejected")
	}
}

func TestSafeHTTPClientBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newSafeHTTPClient(HTTPPolicy{})
	resp, err := client.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("expected loopback request to be blocked, got status %d", resp.StatusCode)
	}
	if !strings.Contains(err.Error(), "ssrf") {
		t.Errorf("expected ssrf error, got: %v", err)
	}
}

func TestSafeHTTPClientBlocksNonHTTPScheme(t *testing.T) {
	client := newSafeHTTPClient(HTTPPolicy{})
	_, err := client.Get("file:///etc/passwd")
	if err == nil {
		t.Fatalf("expected file scheme to be rejected")
	}
}

func TestSafeHTTPClientAllowsAllowedHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// httptest binds to 127.0.0.1, which is loopback; the allowlist must
	// override the IP-based check.
	host := strings.TrimPrefix(srv.URL, "http://")
	client := newSafeHTTPClient(HTTPPolicy{AllowedHosts: []string{host}})
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("allowed host should be reachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
