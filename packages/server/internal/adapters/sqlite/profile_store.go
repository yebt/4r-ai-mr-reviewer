package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
)

// ProfileStore is the SQLite-backed implementation of profile.Repository.
type ProfileStore struct {
	db *sql.DB
}

// NewProfileStore wires a ProfileStore over a database.
func NewProfileStore(db *sql.DB) *ProfileStore {
	return &ProfileStore{db: db}
}

var _ profile.Repository = (*ProfileStore)(nil)

const profileCols = `id, name, language, formality, emojis, samples, style_guide, created_at, updated_at`

// Create inserts a profile row.
func (r *ProfileStore) Create(ctx context.Context, p profile.Profile) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO profiles(`+profileCols+`) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Language, p.Formality, boolToInt(p.Emojis),
		marshalStringList(p.Samples), p.StyleGuide, formatTime(p.CreatedAt), formatTime(p.UpdatedAt))
	if err != nil {
		return fmt.Errorf("profile store: create: %w", err)
	}
	return nil
}

// Save updates a profile's mutable fields.
func (r *ProfileStore) Save(ctx context.Context, p profile.Profile) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET name = ?, language = ?, formality = ?, emojis = ?, samples = ?, style_guide = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.Language, p.Formality, boolToInt(p.Emojis),
		marshalStringList(p.Samples), p.StyleGuide, formatTime(p.UpdatedAt), p.ID)
	if err != nil {
		return fmt.Errorf("profile store: save: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return profile.ErrNotFound
	}
	return nil
}

// Get returns the profile with the given id.
func (r *ProfileStore) Get(ctx context.Context, id string) (profile.Profile, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+profileCols+` FROM profiles WHERE id = ?`, id)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return profile.Profile{}, profile.ErrNotFound
	}
	if err != nil {
		return profile.Profile{}, fmt.Errorf("profile store: get: %w", err)
	}
	return p, nil
}

// List returns all profiles ordered by name.
func (r *ProfileStore) List(ctx context.Context) ([]profile.Profile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+profileCols+` FROM profiles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("profile store: list: %w", err)
	}
	defer rows.Close()

	var out []profile.Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("profile store: scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Delete removes the profile with the given id.
func (r *ProfileStore) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("profile store: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return profile.ErrNotFound
	}
	return nil
}

func scanProfile(s scanner) (profile.Profile, error) {
	var (
		p                  profile.Profile
		emojis             int
		samplesJSON        string
		createdAt, updated string
	)
	if err := s.Scan(&p.ID, &p.Name, &p.Language, &p.Formality, &emojis,
		&samplesJSON, &p.StyleGuide, &createdAt, &updated); err != nil {
		return profile.Profile{}, err
	}
	p.Emojis = emojis != 0
	p.Samples = unmarshalStringList(samplesJSON)
	ct, err := parseTime(createdAt)
	if err != nil {
		return profile.Profile{}, err
	}
	ut, err := parseTime(updated)
	if err != nil {
		return profile.Profile{}, err
	}
	p.CreatedAt = ct
	p.UpdatedAt = ut
	return p, nil
}
