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

// Verify reports whether password unlocks the current vault. Key-file mode has no
// password, so it always succeeds there; password mode returns ErrWrongPassword
// on a mismatch. Used to gate a runtime key change behind the current password.
func (v *Vault) Verify(ctx context.Context, password string) error {
	mode, found, err := v.meta.Get(ctx, metaMode)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotInitialized
	}
	if string(mode) != modePassword {
		return nil
	}
	_, err = v.unlockPassword(ctx, password)
	return err
}

// newKeyMaterial builds the cipher and the app_meta updates for a NEW master key.
// A non-empty password selects password mode (fresh salt + verifier); an empty
// password selects key-file mode and writes a fresh 0600 key file. It does not
// touch stored secrets — the caller re-encrypts them under the returned cipher in
// one transaction (see SecretStore.Rekey).
func (v *Vault) newKeyMaterial(password string) (*crypto.Cipher, map[string][]byte, error) {
	if password == "" {
		key := make([]byte, crypto.KeyLen)
		if _, err := rand.Read(key); err != nil {
			return nil, nil, fmt.Errorf("vault: generate key: %w", err)
		}
		if err := os.WriteFile(v.keyfilePath, key, 0o600); err != nil {
			return nil, nil, fmt.Errorf("vault: write keyfile: %w", err)
		}
		cipher, err := crypto.NewCipher(key)
		if err != nil {
			return nil, nil, err
		}
		return cipher, map[string][]byte{metaMode: []byte(modeKeyfile)}, nil
	}
	salt, err := crypto.NewSalt()
	if err != nil {
		return nil, nil, err
	}
	key, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return nil, nil, err
	}
	cipher, err := crypto.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	verifier, err := cipher.Seal([]byte(verifierText))
	if err != nil {
		return nil, nil, err
	}
	return cipher, map[string][]byte{
		metaSalt:     salt,
		metaVerifier: verifier,
		metaMode:     []byte(modePassword),
	}, nil
}

// Rekeyer re-encrypts every stored secret under a new cipher and applies the
// given app_meta updates atomically, then swaps its live cipher. SecretStore
// implements it; the interface keeps this package free of the sqlite adapter.
type Rekeyer interface {
	Rekey(ctx context.Context, newCipher *crypto.Cipher, meta map[string][]byte) error
}

// Service coordinates a runtime master-key change: it verifies the current
// password, derives the new key material, and hands both to the Rekeyer so every
// secret is re-encrypted in one transaction.
type Service struct {
	vault   *Vault
	secrets Rekeyer
}

// NewService wires the vault Service over a Vault and a Rekeyer (the secret store).
func NewService(v *Vault, secrets Rekeyer) *Service {
	return &Service{vault: v, secrets: secrets}
}

// Status reports whether the vault is initialized and password-protected.
func (s *Service) Status(ctx context.Context) (Status, error) {
	return s.vault.Status(ctx)
}

// ChangeSecret rotates the master key. oldPassword must unlock the current vault
// (ignored in key-file mode). An empty newPassword switches to key-file mode
// (rotating the key file); a non-empty newPassword sets/changes the password.
// Every secret is re-encrypted under the new key in one transaction; on any
// failure nothing changes. Returns the resulting vault status.
//
// Operational note for the caller: after switching to (or changing) a password,
// the boot-time AIR_PASSWORD must be updated to the new value before the next
// restart, or the vault will not unlock. Switching to key-file mode needs no
// config change.
func (s *Service) ChangeSecret(ctx context.Context, oldPassword, newPassword string) (Status, error) {
	if err := s.vault.Verify(ctx, oldPassword); err != nil {
		return Status{}, err
	}
	newCipher, meta, err := s.vault.newKeyMaterial(newPassword)
	if err != nil {
		return Status{}, err
	}
	if err := s.secrets.Rekey(ctx, newCipher, meta); err != nil {
		return Status{}, err
	}
	return s.vault.Status(ctx)
}
