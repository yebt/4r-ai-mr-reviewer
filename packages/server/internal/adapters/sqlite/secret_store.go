package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
)

// SecretStore is the SQLite-backed, encrypted implementation of secret.Store.
type SecretStore struct {
	db     *sql.DB
	cipher *crypto.Cipher
}

// NewSecretStore wires a SecretStore over a database and cipher.
func NewSecretStore(db *sql.DB, cipher *crypto.Cipher) *SecretStore {
	return &SecretStore{db: db, cipher: cipher}
}

var _ secret.Store = (*SecretStore)(nil)

// Set encrypts value and upserts it under name.
func (s *SecretStore) Set(ctx context.Context, name string, value []byte) error {
	blob, err := s.cipher.Seal(value)
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
	value, err := s.cipher.Open(blob)
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
