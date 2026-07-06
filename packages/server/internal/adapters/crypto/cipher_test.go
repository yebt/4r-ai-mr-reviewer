package crypto

import (
	"bytes"
	"testing"
)

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	key, err := DeriveKey("correct horse battery staple", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestSealOpenRoundTrip(t *testing.T) {
	c := newTestCipher(t)
	plaintext := []byte("glpat-super-secret-token")

	blob, err := c.Seal(plaintext)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(blob, plaintext) {
		t.Fatal("ciphertext must not contain the plaintext")
	}

	got, err := c.Open(blob)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestSealProducesDistinctCiphertexts(t *testing.T) {
	c := newTestCipher(t)
	a, _ := c.Seal([]byte("same"))
	b, _ := c.Seal([]byte("same"))
	if bytes.Equal(a, b) {
		t.Fatal("nonce reuse: two seals of the same plaintext produced identical output")
	}
}

func TestOpenWithWrongKeyFails(t *testing.T) {
	c := newTestCipher(t)
	blob, _ := c.Seal([]byte("secret"))

	otherSalt, _ := NewSalt()
	otherKey, _ := DeriveKey("different password", otherSalt)
	other, _ := NewCipher(otherKey)

	if _, err := other.Open(blob); err == nil {
		t.Fatal("expected authentication failure opening with the wrong key")
	}
}

func TestDeriveKeyIsDeterministic(t *testing.T) {
	salt, _ := NewSalt()
	k1, _ := DeriveKey("pw", salt)
	k2, _ := DeriveKey("pw", salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("DeriveKey must be deterministic for the same password and salt")
	}
	if len(k1) != KeyLen {
		t.Fatalf("derived key length = %d, want %d", len(k1), KeyLen)
	}
}

func TestNewCipherRejectsBadKeyLength(t *testing.T) {
	if _, err := NewCipher([]byte("too short")); err == nil {
		t.Fatal("expected error for non-32-byte key")
	}
}

func TestOpenRejectsShortBlob(t *testing.T) {
	c := newTestCipher(t)
	if _, err := c.Open([]byte{0x01}); err == nil {
		t.Fatal("expected ErrCiphertextTooShort")
	}
}
