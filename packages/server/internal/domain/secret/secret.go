// Package secret defines the domain contract for storing sensitive values
// (API keys, tokens) encrypted at rest. The domain never handles ciphertext;
// adapters are responsible for encryption.
package secret

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a secret does not exist.
var ErrNotFound = errors.New("secret: not found")

// Info describes a stored secret without ever exposing its value. It is what
// the UI shows so the user knows whether a token is present.
type Info struct {
	Name   string
	Stored bool
}

// Store persists and retrieves encrypted secret values. Implementations must
// encrypt at rest; List and Info must never leak plaintext.
type Store interface {
	Set(ctx context.Context, name string, value []byte) error
	Get(ctx context.Context, name string) ([]byte, error)
	Exists(ctx context.Context, name string) (bool, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]Info, error)
}
