package auth

import (
	"strings"
	"testing"
	"time"
)

const testLifetime = time.Hour

func TestEnabled(t *testing.T) {
	if NewManager("", testLifetime).Enabled() {
		t.Fatal("empty password: Enabled() = true, want false")
	}
	if !NewManager("pw", testLifetime).Enabled() {
		t.Fatal("non-empty password: Enabled() = false, want true")
	}
}

func TestIssueVerifyRoundTrip(t *testing.T) {
	m := NewManager("pw", testLifetime)
	if !m.Verify(m.Issue()) {
		t.Fatal("freshly issued token failed to verify")
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	m := NewManager("pw", testLifetime)
	token := m.Issue()
	parts := strings.Split(token, ".")
	// Re-sign is impossible without the key, so swap the payload for a different
	// (validly encoded) one; the signature no longer matches.
	tampered := payload64("9999999999") + "." + parts[1]
	if m.Verify(tampered) {
		t.Fatal("tampered payload verified, want rejection")
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	m := NewManager("pw", testLifetime)
	parts := strings.Split(m.Issue(), ".")
	tampered := parts[0] + "." + payload64("not-a-real-signature")
	if m.Verify(tampered) {
		t.Fatal("tampered signature verified, want rejection")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	m := NewManager("pw", testLifetime)
	for _, v := range []string{"", "nodot", "a.b.c", "!!!.###", "onlyone.", "."} {
		if m.Verify(v) {
			t.Fatalf("malformed token %q verified, want rejection", v)
		}
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	m := NewManager("pw", testLifetime)
	// Craft a token issued two lifetimes ago: its expiry is one lifetime in the
	// past. No sleeping required.
	expired := m.issueAt(time.Now().Add(-2 * testLifetime))
	if m.Verify(expired) {
		t.Fatal("expired token verified, want rejection")
	}
}

func TestVerifyRejectsAtExactExpiry(t *testing.T) {
	m := NewManager("pw", testLifetime)
	// A token whose expiry is exactly now: issued one lifetime in the past so
	// expiry = (now - lifetime) + lifetime = now. Verify compares with a strict
	// `<`, so a token expiring at the current second must be rejected. This pins
	// the boundary against a `<=` regression.
	atExpiry := m.issueAt(time.Now().Add(-testLifetime))
	if m.Verify(atExpiry) {
		t.Fatal("token at exact expiry verified, want rejection (expiry is strict <)")
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	issuer := NewManager("pw", testLifetime)
	other := NewManager("different", testLifetime)
	if other.Verify(issuer.Issue()) {
		t.Fatal("token verified under a different password's key, want rejection")
	}
}

func TestCheckPassword(t *testing.T) {
	m := NewManager("s3cret", testLifetime)
	if !m.CheckPassword("s3cret") {
		t.Fatal("correct password rejected")
	}
	if m.CheckPassword("wrong") {
		t.Fatal("wrong password accepted")
	}
	if m.CheckPassword("") {
		t.Fatal("empty candidate accepted")
	}
}
