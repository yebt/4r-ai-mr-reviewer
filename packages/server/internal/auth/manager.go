// Package auth issues and verifies stateless, signed session tokens for the
// HTTP API. It holds no HTTP types on purpose: the middleware and handlers in
// internal/http adapt this to cookies.
//
// A token is a bearer of an expiry only. Its value is
//
//	base64url(payload) + "." + base64url(HMAC-SHA256(payload, key))
//
// where payload is the token's unix-seconds expiry and key = sha256(password).
// There is no server-side session table: verification recomputes the HMAC in
// constant time and rejects anything tampered, unparseable, or expired.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// Manager signs and verifies session tokens for a single configured password.
type Manager struct {
	password string
	key      []byte
	lifetime time.Duration
}

// NewManager builds a Manager for the given password and session lifetime. An
// empty password yields a disabled Manager (see Enabled). The signing key is
// derived deterministically as sha256(password) so tokens survive restarts
// without any stored state.
func NewManager(password string, lifetime time.Duration) *Manager {
	sum := sha256.Sum256([]byte(password))
	return &Manager{password: password, key: sum[:], lifetime: lifetime}
}

// Enabled reports whether authentication is configured (non-empty password).
func (m *Manager) Enabled() bool { return m.password != "" }

// Lifetime is the session lifetime tokens are issued with.
func (m *Manager) Lifetime() time.Duration { return m.lifetime }

// Issue signs a fresh token expiring lifetime from now.
func (m *Manager) Issue() string { return m.issueAt(time.Now()) }

// issueAt signs a token whose expiry is at+lifetime. It is the internal seam
// tests use to craft tokens with explicit (including past) expiries without
// sleeping.
func (m *Manager) issueAt(at time.Time) string {
	payload := strconv.FormatInt(at.Add(m.lifetime).Unix(), 10)
	return payload64(payload) + "." + payload64(string(m.sign(payload)))
}

// Verify reports whether value is a well-formed, untampered, unexpired token
// signed with this Manager's key. It returns false on any parse, signature, or
// expiry failure.
func (m *Manager) Verify(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	// Constant-time signature check before trusting any payload byte.
	if !hmac.Equal(gotSig, m.sign(string(payload))) {
		return false
	}
	expiry, err := strconv.ParseInt(string(payload), 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expiry
}

// CheckPassword compares a candidate against the configured password in
// constant time. Both operands are hashed to a fixed 32-byte digest first so the
// comparison length never depends on the password length: subtle.ConstantTimeCompare
// short-circuits on a length mismatch, which would otherwise leak the password
// length through timing.
func (m *Manager) CheckPassword(candidate string) bool {
	got := sha256.Sum256([]byte(candidate))
	want := sha256.Sum256([]byte(m.password))
	return subtle.ConstantTimeCompare(got[:], want[:]) == 1
}

// sign returns the HMAC-SHA256 of payload under the Manager's key.
func (m *Manager) sign(payload string) []byte {
	mac := hmac.New(sha256.New, m.key)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// payload64 is base64url without padding, the encoding used for both token
// segments.
func payload64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
