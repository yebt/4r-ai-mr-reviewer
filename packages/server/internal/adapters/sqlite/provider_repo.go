package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
)

// ProviderRepo is the SQLite-backed implementation of provider.Repository.
type ProviderRepo struct {
	db *sql.DB
}

// NewProviderRepo wires a ProviderRepo over a database.
func NewProviderRepo(db *sql.DB) *ProviderRepo {
	return &ProviderRepo{db: db}
}

var _ provider.Repository = (*ProviderRepo)(nil)

const providerCols = `id, name, kind, base_url, model, api_key_ref, is_default, created_at`

// Create inserts a provider row.
func (r *ProviderRepo) Create(ctx context.Context, p provider.Provider) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO providers(`+providerCols+`) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, string(p.Kind), p.BaseURL, p.Model, p.APIKeyRef, boolToInt(p.IsDefault), formatTime(p.CreatedAt))
	if err != nil {
		return fmt.Errorf("provider repo: create: %w", err)
	}
	return nil
}

// Get returns the provider with the given id.
func (r *ProviderRepo) Get(ctx context.Context, id string) (provider.Provider, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+providerCols+` FROM providers WHERE id = ?`, id)
	p, err := scanProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return provider.Provider{}, provider.ErrNotFound
	}
	if err != nil {
		return provider.Provider{}, fmt.Errorf("provider repo: get: %w", err)
	}
	return p, nil
}

// List returns all providers ordered by name.
func (r *ProviderRepo) List(ctx context.Context) ([]provider.Provider, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+providerCols+` FROM providers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("provider repo: list: %w", err)
	}
	defer rows.Close()

	var out []provider.Provider
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, fmt.Errorf("provider repo: scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Delete removes the provider with the given id.
func (r *ProviderRepo) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id); err != nil {
		return fmt.Errorf("provider repo: delete: %w", err)
	}
	return nil
}

// SetDefault marks id as the sole default, clearing any previous default in
// the same transaction so the single-default index is never violated.
func (r *ProviderRepo) SetDefault(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("provider repo: set default: begin: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE providers SET is_default = 0 WHERE is_default = 1`)
	if err != nil {
		return fmt.Errorf("provider repo: clear default: %w", err)
	}
	_ = res

	res, err = tx.ExecContext(ctx, `UPDATE providers SET is_default = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("provider repo: mark default: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return provider.ErrNotFound
	}
	return tx.Commit()
}

// GetDefault returns the default provider, or ErrNotFound if none is set.
func (r *ProviderRepo) GetDefault(ctx context.Context) (provider.Provider, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+providerCols+` FROM providers WHERE is_default = 1`)
	p, err := scanProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return provider.Provider{}, provider.ErrNotFound
	}
	if err != nil {
		return provider.Provider{}, fmt.Errorf("provider repo: get default: %w", err)
	}
	return p, nil
}

func scanProvider(s scanner) (provider.Provider, error) {
	var (
		p         provider.Provider
		kind      string
		isDefault int
		createdAt string
	)
	if err := s.Scan(&p.ID, &p.Name, &kind, &p.BaseURL, &p.Model, &p.APIKeyRef, &isDefault, &createdAt); err != nil {
		return provider.Provider{}, err
	}
	p.Kind = provider.Kind(kind)
	p.IsDefault = isDefault != 0
	t, err := parseTime(createdAt)
	if err != nil {
		return provider.Provider{}, err
	}
	p.CreatedAt = t
	return p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
