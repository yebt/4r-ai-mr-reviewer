// Package crypto provides encryption of secret values at rest.
//
// Secrets are sealed with AES-256-GCM. The 32-byte key is derived from the
// user's optional app password via PBKDF2-HMAC-SHA256. When no password is
// set, the key is generated once and persisted to a 0600 key file: still
// encrypted at rest, but the key lives beside the data (see docs/DESIGN.md).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

const (
	// KeyLen is the AES-256 key length in bytes.
	KeyLen = 32
	// SaltLen is the length of the KDF salt in bytes.
	SaltLen = 16
	// pbkdf2Iterations follows OWASP 2023 guidance for PBKDF2-HMAC-SHA256.
	pbkdf2Iterations = 600_000
)

// ErrCiphertextTooShort is returned when a blob is smaller than the GCM nonce.
var ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")

// DeriveKey derives a 32-byte key from a password and salt using
// PBKDF2-HMAC-SHA256. The same password and salt always yield the same key.
func DeriveKey(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, KeyLen)
}

// NewSalt returns a fresh cryptographically random salt.
func NewSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Cipher seals and opens secret values with a fixed AES-256-GCM key.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher builds a Cipher from a 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != KeyLen {
		return nil, fmt.Errorf("crypto: key must be %d bytes, got %d", KeyLen, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Seal encrypts plaintext. The returned blob is nonce || ciphertext.
func (c *Cipher) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Open decrypts a nonce || ciphertext blob produced by Seal.
func (c *Cipher) Open(blob []byte) ([]byte, error) {
	ns := c.aead.NonceSize()
	if len(blob) < ns {
		return nil, ErrCiphertextTooShort
	}
	nonce, ciphertext := blob[:ns], blob[ns:]
	return c.aead.Open(nil, nonce, ciphertext, nil)
}
