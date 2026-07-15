package netutil

import (
	"errors"
	"net"
	"testing"
)

// withLookupIP swaps the package-level resolver for the duration of a test.
func withLookupIP(t *testing.T, fn func(host string) ([]net.IP, error)) {
	t.Helper()
	orig := lookupIP
	lookupIP = fn
	t.Cleanup(func() { lookupIP = orig })
}

func TestRequireSecureBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"empty allowed", "", false},
		{"https public allowed", "https://api.openai.com/v1", false},
		{"https private allowed", "https://192.168.1.10", false},
		{"http localhost allowed", "http://localhost:1234/v1", false},
		{"http localhost uppercase allowed", "http://LOCALHOST:1234", false},
		{"http loopback ip allowed", "http://127.0.0.1:8080", false},
		{"http private 10 allowed", "http://10.0.0.5:11434/v1", false},
		{"http private 192 allowed", "http://192.168.1.20", false},
		{"http private 172 allowed", "http://172.16.4.4:8000", false},
		{"http link-local allowed", "http://169.254.1.1", false},
		{"http ipv6 loopback allowed", "http://[::1]:8080", false},
		{"http public ip rejected", "http://8.8.8.8:80", true},
		{"bad scheme rejected", "ftp://example.com", true},
		{"scheme-less rejected", "neu", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireSecureBaseURL(tc.raw)
			if tc.wantErr != (err != nil) {
				t.Fatalf("RequireSecureBaseURL(%q) err = %v, wantErr = %v", tc.raw, err, tc.wantErr)
			}
		})
	}
}

func TestRequireSecureBaseURL_HTTPPublicHostnameRejected(t *testing.T) {
	withLookupIP(t, func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	})
	if err := RequireSecureBaseURL("http://public.example.com/v1"); err == nil {
		t.Fatal("expected rejection for http hostname resolving to a public IP")
	}
}

func TestRequireSecureBaseURL_HTTPPrivateHostnameAllowed(t *testing.T) {
	withLookupIP(t, func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.1.2.3")}, nil
	})
	if err := RequireSecureBaseURL("http://gitlab.internal/api"); err != nil {
		t.Fatalf("expected private-resolving hostname to be allowed, got %v", err)
	}
}

func TestRequireSecureBaseURL_HTTPMixedResolutionRejected(t *testing.T) {
	withLookupIP(t, func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.1.2.3"), net.ParseIP("8.8.8.8")}, nil
	})
	if err := RequireSecureBaseURL("http://mixed.example"); err == nil {
		t.Fatal("expected rejection when any resolved IP is public")
	}
}

func TestRequireSecureBaseURL_HTTPResolutionFailureRejected(t *testing.T) {
	withLookupIP(t, func(string) ([]net.IP, error) {
		return nil, errors.New("no such host")
	})
	if err := RequireSecureBaseURL("http://unresolvable.example"); err == nil {
		t.Fatal("expected rejection when hostname does not resolve")
	}
}
