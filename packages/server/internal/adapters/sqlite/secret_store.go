package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
)

// SecretStore is the SQLite-backed, encrypted implementation of secret.Store.
// The cipher can be rotated at runtime (Rekey); mu guards the swap against
// concurrent Set/Get so a re-encryption never interleaves with a plain op.
type SecretStore struct {
	db     *sql.DB
	mu     sync.RWMutex
	cipher *crypto.Cipher
}

// NewSecretStore wires a SecretStore over a database and cipher.
func NewSecretStore(db *sql.DB, cipher *crypto.Cipher) *SecretStore {
	return &SecretStore{db: db, cipher: cipher}
}

var _ secret.Store = (*SecretStore)(nil)

// currentCipher returns the live cipher under a read lock, so ordinary Set/Get
// run concurrently with each other but never during a Rekey swap.
func (s *SecretStore) currentCipher() *crypto.Cipher {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cipher
}

// Set encrypts value and upserts it under name.
func (s *SecretStore) Set(ctx context.Context, name string, value []byte) error {
	blob, err := s.currentCipher().Seal(value)
	if err != nil {
		return fmt.Errorf("secret store: seal %q: %w", name, err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO secrets(name, ciphertext) VALUES(?, ?)
		ON CONFLICT(name) DO UPDATE SET
			ciphertext = excluded.ciphertext,
			updated_at = CURRENT_TIMESTAMP`,
		name, blob)
	if err != nil {
		return fmt.Errorf("secret store: set %q: %w", name, err)
	}
	return nil
}

// Get decrypts and returns the value stored under name.
func (s *SecretStore) Get(ctx context.Context, name string) ([]byte, error) {
	var blob []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT ciphertext FROM secrets WHERE name = ?`, name,
	).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, secret.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("secret store: get %q: %w", name, err)
	}
	value, err := s.currentCipher().Open(blob)
	if err != nil {
		return nil, fmt.Errorf("secret store: open %q: %w", name, err)
	}
	return value, nil
}

// Exists reports whether a secret is stored under name.
func (s *SecretStore) Exists(ctx context.Context, name string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM secrets WHERE name = ?`, name,
	).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("secret store: exists %q: %w", name, err)
	}
	return true, nil
}

// Delete removes the secret under name. Deleting a missing secret is a no-op.
func (s *SecretStore) Delete(ctx context.Context, name string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM secrets WHERE name = ?`, name); err != nil {
		return fmt.Errorf("secret store: delete %q: %w", name, err)
	}
	return nil
}

// List returns the names of stored secrets, never their values.
func (s *SecretStore) List(ctx context.Context) ([]secret.Info, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM secrets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("secret store: list: %w", err)
	}
	defer rows.Close()

	var infos []secret.Info
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("secret store: scan: %w", err)
		}
		infos = append(infos, secret.Info{Name: name, Stored: true})
	}
	return infos, rows.Err()
}

// Rekey re-encrypts every stored secret from the current cipher to newCipher and
// applies the given app_meta updates (KDF salt, verifier, mode) in ONE
// transaction. It takes the write lock for the whole operation, so no Set/Get can
// interleave: either the whole vault flips to the new key (committed, then the
// live cipher swaps) or nothing changes (any decrypt/DB failure rolls back and
// the old cipher stays live). Passing an empty meta map re-encrypts under a new
// key without changing the stored key material description (used when only the
// key rotates, e.g. key-file mode).
func (s *SecretStore) Rekey(ctx context.Context, newCipher *crypto.Cipher, meta map[string][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.cipher

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("secret store: rekey begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	// Decrypt every secret with the OLD cipher first, buffering plaintext, so a
	// wrong old key fails before any row is written.
	rows, err := tx.QueryContext(ctx, `SELECT name, ciphertext FROM secrets`)
	if err != nil {
		return fmt.Errorf("secret store: rekey list: %w", err)
	}
	type item struct {
		name  string
		plain []byte
	}
	var items []item
	for rows.Next() {
		var name string
		var blob []byte
		if err := rows.Scan(&name, &blob); err != nil {
			rows.Close()
			return fmt.Errorf("secret store: rekey scan: %w", err)
		}
		plain, err := old.Open(blob)
		if err != nil {
			rows.Close()
			return fmt.Errorf("secret store: rekey decrypt %q: %w", name, err)
		}
		items = append(items, item{name: name, plain: plain})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("secret store: rekey rows: %w", err)
	}
	rows.Close()

	for _, it := range items {
		blob, err := newCipher.Seal(it.plain)
		if err != nil {
			return fmt.Errorf("secret store: rekey seal %q: %w", it.name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE secrets SET ciphertext = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?`,
			blob, it.name); err != nil {
			return fmt.Errorf("secret store: rekey update %q: %w", it.name, err)
		}
	}
	for k, v := range meta {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO app_meta(key, value) VALUES(?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return fmt.Errorf("secret store: rekey meta %q: %w", k, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("secret store: rekey commit: %w", err)
	}
	// Persisted. Swap the live cipher so subsequent ops use the new key.
	s.cipher = newCipher
	return nil
}
