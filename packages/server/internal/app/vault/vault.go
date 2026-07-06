// Package vault manages the lifecycle of the master key used to encrypt
// secrets at rest. The key comes from either the user's app password
// (PBKDF2) or, when no password is set, a 0600 key file beside the database.
package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
)

// Sentinel errors.
var (
	ErrNotInitialized = errors.New("vault: not initialized")
	ErrAlreadyInit    = errors.New("vault: already initialized")
	ErrWrongPassword  = errors.New("vault: wrong password")
)

const (
	metaMode     = "vault.mode"
	metaSalt     = "vault.salt"
	metaVerifier = "vault.verifier"
	modePassword = "password"
	modeKeyfile  = "keyfile"
	// verifierText is sealed at setup and re-opened at unlock to confirm the
	// derived key (and therefore the password) is correct.
	verifierText = "ai-reviewer-vault-v1"
)

// MetaStore is the small persistence port the vault needs.
type MetaStore interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte) error
}

// Vault sets up and unlocks the secret cipher.
type Vault struct {
	meta        MetaStore
	keyfilePath string
}

// New wires a Vault over a MetaStore and the key-file location.
func New(meta MetaStore, keyfilePath string) *Vault {
	return &Vault{meta: meta, keyfilePath: keyfilePath}
}

// Status describes whether the vault is initialized and password-protected.
type Status struct {
	Initialized       bool
	PasswordProtected bool
}

// Status reports the current vault state.
func (v *Vault) Status(ctx context.Context) (Status, error) {
	mode, found, err := v.meta.Get(ctx, metaMode)
	if err != nil {
		return Status{}, err
	}
	if !found {
		return Status{}, nil
	}
	return Status{Initialized: true, PasswordProtected: string(mode) == modePassword}, nil
}

// Initialize sets up the vault and returns the unlocked cipher. An empty
// password selects key-file mode. It fails if the vault already exists.
func (v *Vault) Initialize(ctx context.Context, password string) (*crypto.Cipher, error) {
	st, err := v.Status(ctx)
	if err != nil {
		return nil, err
	}
	if st.Initialized {
		return nil, ErrAlreadyInit
	}
	if password == "" {
		return v.initKeyfile(ctx)
	}
	return v.initPassword(ctx, password)
}

func (v *Vault) initPassword(ctx context.Context, password string) (*crypto.Cipher, error) {
	salt, err := crypto.NewSalt()
	if err != nil {
		return nil, err
	}
	key, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	cipher, err := crypto.NewCipher(key)
	if err != nil {
		return nil, err
	}
	verifier, err := cipher.Seal([]byte(verifierText))
	if err != nil {
		return nil, err
	}
	if err := v.meta.Set(ctx, metaSalt, salt); err != nil {
		return nil, err
	}
	if err := v.meta.Set(ctx, metaVerifier, verifier); err != nil {
		return nil, err
	}
	if err := v.meta.Set(ctx, metaMode, []byte(modePassword)); err != nil {
		return nil, err
	}
	return cipher, nil
}

func (v *Vault) initKeyfile(ctx context.Context) (*crypto.Cipher, error) {
	key := make([]byte, crypto.KeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("vault: generate key: %w", err)
	}
	if err := os.WriteFile(v.keyfilePath, key, 0o600); err != nil {
		return nil, fmt.Errorf("vault: write keyfile: %w", err)
	}
	cipher, err := crypto.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if err := v.meta.Set(ctx, metaMode, []byte(modeKeyfile)); err != nil {
		return nil, err
	}
	return cipher, nil
}

// Unlock returns the cipher. For password mode the password must be correct;
// for key-file mode the password argument is ignored.
func (v *Vault) Unlock(ctx context.Context, password string) (*crypto.Cipher, error) {
	mode, found, err := v.meta.Get(ctx, metaMode)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotInitialized
	}
	switch string(mode) {
	case modeKeyfile:
		return v.unlockKeyfile()
	case modePassword:
		return v.unlockPassword(ctx, password)
	default:
		return nil, fmt.Errorf("vault: unknown mode %q", mode)
	}
}

func (v *Vault) unlockPassword(ctx context.Context, password string) (*crypto.Cipher, error) {
	salt, ok, err := v.meta.Get(ctx, metaSalt)
	if err != nil {
		return nil, err
	}
	verifier, ok2, err := v.meta.Get(ctx, metaVerifier)
	if err != nil {
		return nil, err
	}
	if !ok || !ok2 {
		return nil, ErrNotInitialized
	}
	key, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	cipher, err := crypto.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain, err := cipher.Open(verifier)
	if err != nil || !bytes.Equal(plain, []byte(verifierText)) {
		return nil, ErrWrongPassword
	}
	return cipher, nil
}

func (v *Vault) unlockKeyfile() (*crypto.Cipher, error) {
	key, err := os.ReadFile(v.keyfilePath)
	if err != nil {
		return nil, fmt.Errorf("vault: read keyfile: %w", err)
	}
	return crypto.NewCipher(key)
}
