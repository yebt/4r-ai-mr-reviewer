// Package id generates short random identifiers for domain entities.
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a 32-character hex identifier backed by 16 random bytes.
func New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read never fails on supported platforms; if it does,
		// there is no safe way to continue generating identifiers.
		panic("id: entropy source failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
