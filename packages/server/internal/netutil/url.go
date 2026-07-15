// Package netutil holds small, dependency-free network helpers shared across
// the application layer.
package netutil

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// lookupIP resolves a hostname to its IP addresses. It is a package-level var so
// tests can stub DNS resolution without threading a resolver through services.
var lookupIP = net.LookupIP

// RequireSecureBaseURL rejects cleartext base URLs that would leak credentials
// (API keys, GitLab PATs) to a public host over the network.
//
// Policy:
//   - Empty string is allowed: BaseURL is optional and a provider default is used.
//   - The URL must parse and use the http or https scheme.
//   - https is always allowed, for any host.
//   - http is allowed ONLY when the host is local/private:
//   - the hostname "localhost" (case-insensitive), or
//   - an IP literal that is loopback, private, or link-local, or
//   - a non-IP hostname that resolves AND every resolved address is
//     loopback, private, or link-local. If it resolves to any public
//     address, or resolution fails, the http URL is rejected.
func RequireSecureBaseURL(raw string) error {
	if raw == "" {
		return nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid base URL")
	}

	switch u.Scheme {
	case "https":
		return nil
	case "http":
		// Fall through to the private-host check below.
	default:
		return fmt.Errorf("base URL must use http or https")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("invalid base URL")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if isLocalOrPrivate(ip) {
			return nil
		}
		return errPublicHTTP()
	}

	ips, err := lookupIP(host)
	if err != nil || len(ips) == 0 {
		return errPublicHTTP()
	}
	for _, ip := range ips {
		if !isLocalOrPrivate(ip) {
			return errPublicHTTP()
		}
	}
	return nil
}

// isLocalOrPrivate reports whether ip belongs to a loopback, private, or
// link-local range and is therefore safe to reach over cleartext http.
func isLocalOrPrivate(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func errPublicHTTP() error {
	return fmt.Errorf("base URL must use https for a public host (http is allowed only for localhost or a private network)")
}
