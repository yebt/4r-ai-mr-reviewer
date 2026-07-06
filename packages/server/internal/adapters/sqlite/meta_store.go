package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// MetaStore reads and writes single rows in app_meta (KDF salt, verifier, etc.).
type MetaStore struct {
	db *sql.DB
}

// NewMetaStore wires a MetaStore over a database.
func NewMetaStore(db *sql.DB) *MetaStore {
	return &MetaStore{db: db}
}

// Get returns the value for key. found is false when the key is absent.
func (m *MetaStore) Get(ctx context.Context, key string) (value []byte, found bool, err error) {
	err = m.db.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("meta store: get %q: %w", key, err)
	}
	return value, true, nil
}

// Set upserts the value for key.
func (m *MetaStore) Set(ctx context.Context, key string, value []byte) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO app_meta(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("meta store: set %q: %w", key, err)
	}
	return nil
}
