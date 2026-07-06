package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
)

// RepoStore is the SQLite-backed implementation of repo.Repository.
type RepoStore struct {
	db *sql.DB
}

// NewRepoStore wires a RepoStore over a database.
func NewRepoStore(db *sql.DB) *RepoStore {
	return &RepoStore{db: db}
}

var _ repo.Repository = (*RepoStore)(nil)

const repoCols = `id, name, url, account_id, provider_id, model, created_at`

// Create inserts a repo row.
func (r *RepoStore) Create(ctx context.Context, x repo.Repo) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO repos(`+repoCols+`) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		x.ID, x.Name, x.URL, x.AccountID, nullString(x.ProviderID), x.Model, formatTime(x.CreatedAt))
	if err != nil {
		return fmt.Errorf("repo store: create: %w", err)
	}
	return nil
}

// Get returns the repo with the given id.
func (r *RepoStore) Get(ctx context.Context, id string) (repo.Repo, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+repoCols+` FROM repos WHERE id = ?`, id)
	x, err := scanRepo(row)
	if errors.Is(err, sql.ErrNoRows) {
		return repo.Repo{}, repo.ErrNotFound
	}
	if err != nil {
		return repo.Repo{}, fmt.Errorf("repo store: get: %w", err)
	}
	return x, nil
}

// List returns all repos ordered by name.
func (r *RepoStore) List(ctx context.Context) ([]repo.Repo, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+repoCols+` FROM repos ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("repo store: list: %w", err)
	}
	defer rows.Close()

	var out []repo.Repo
	for rows.Next() {
		x, err := scanRepo(rows)
		if err != nil {
			return nil, fmt.Errorf("repo store: scan: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Update changes the mutable fields (name, provider assignment, model).
func (r *RepoStore) Update(ctx context.Context, x repo.Repo) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE repos SET name = ?, provider_id = ?, model = ? WHERE id = ?`,
		x.Name, nullString(x.ProviderID), x.Model, x.ID)
	if err != nil {
		return fmt.Errorf("repo store: update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes the repo with the given id.
func (r *RepoStore) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM repos WHERE id = ?`, id); err != nil {
		return fmt.Errorf("repo store: delete: %w", err)
	}
	return nil
}

func scanRepo(s scanner) (repo.Repo, error) {
	var (
		x          repo.Repo
		providerID sql.NullString
		createdAt  string
	)
	if err := s.Scan(&x.ID, &x.Name, &x.URL, &x.AccountID, &providerID, &x.Model, &createdAt); err != nil {
		return repo.Repo{}, err
	}
	x.ProviderID = providerID.String
	t, err := parseTime(createdAt)
	if err != nil {
		return repo.Repo{}, err
	}
	x.CreatedAt = t
	return x, nil
}

// nullString maps "" to SQL NULL so an empty provider assignment is stored as
// NULL (enabling ON DELETE SET NULL).
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
