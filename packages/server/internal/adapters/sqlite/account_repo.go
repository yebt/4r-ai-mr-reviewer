package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
)

// AccountRepo is the SQLite-backed implementation of account.Repository.
type AccountRepo struct {
	db *sql.DB
}

// NewAccountRepo wires an AccountRepo over a database.
func NewAccountRepo(db *sql.DB) *AccountRepo {
	return &AccountRepo{db: db}
}

var _ account.Repository = (*AccountRepo)(nil)

// Create inserts an account row.
func (r *AccountRepo) Create(ctx context.Context, a account.Account) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO accounts(id, name, base_url, token_ref, created_at) VALUES(?, ?, ?, ?, ?)`,
		a.ID, a.Name, a.BaseURL, a.TokenRef, formatTime(a.CreatedAt))
	if err != nil {
		return fmt.Errorf("account repo: create: %w", err)
	}
	return nil
}

// Get returns the account with the given id.
func (r *AccountRepo) Get(ctx context.Context, id string) (account.Account, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, base_url, token_ref, created_at FROM accounts WHERE id = ?`, id)
	a, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return account.Account{}, account.ErrNotFound
	}
	if err != nil {
		return account.Account{}, fmt.Errorf("account repo: get: %w", err)
	}
	return a, nil
}

// List returns all accounts ordered by name.
func (r *AccountRepo) List(ctx context.Context) ([]account.Account, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, base_url, token_ref, created_at FROM accounts ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("account repo: list: %w", err)
	}
	defer rows.Close()

	var out []account.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("account repo: scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Delete removes the account with the given id. Removing a missing account is
// a no-op.
func (r *AccountRepo) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id); err != nil {
		return fmt.Errorf("account repo: delete: %w", err)
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanAccount(s scanner) (account.Account, error) {
	var (
		a         account.Account
		createdAt string
	)
	if err := s.Scan(&a.ID, &a.Name, &a.BaseURL, &a.TokenRef, &createdAt); err != nil {
		return account.Account{}, err
	}
	t, err := parseTime(createdAt)
	if err != nil {
		return account.Account{}, err
	}
	a.CreatedAt = t
	return a, nil
}
