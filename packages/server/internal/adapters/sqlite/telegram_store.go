package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
)

// TelegramStore is the SQLite-backed implementation of telegram.Repository.
type TelegramStore struct {
	db *sql.DB
}

// NewTelegramStore wires a TelegramStore over a database.
func NewTelegramStore(db *sql.DB) *TelegramStore {
	return &TelegramStore{db: db}
}

var _ telegram.Repository = (*TelegramStore)(nil)

const telegramCols = `id, name, chat_id, thread_id, token_ref, is_default, is_bot, created_at`

// Create inserts a telegram target row.
func (r *TelegramStore) Create(ctx context.Context, t telegram.Target) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO telegram_targets(`+telegramCols+`) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.ChatID, t.ThreadID, t.TokenRef, boolToInt(t.IsDefault), boolToInt(t.IsBot), formatTime(t.CreatedAt))
	if err != nil {
		return fmt.Errorf("telegram store: create: %w", err)
	}
	return nil
}

// Get returns the telegram target with the given id.
func (r *TelegramStore) Get(ctx context.Context, id string) (telegram.Target, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+telegramCols+` FROM telegram_targets WHERE id = ?`, id)
	t, err := scanTelegram(row)
	if errors.Is(err, sql.ErrNoRows) {
		return telegram.Target{}, telegram.ErrNotFound
	}
	if err != nil {
		return telegram.Target{}, fmt.Errorf("telegram store: get: %w", err)
	}
	return t, nil
}

// List returns all telegram targets ordered by name.
func (r *TelegramStore) List(ctx context.Context) ([]telegram.Target, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+telegramCols+` FROM telegram_targets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("telegram store: list: %w", err)
	}
	defer rows.Close()

	var out []telegram.Target
	for rows.Next() {
		t, err := scanTelegram(rows)
		if err != nil {
			return nil, fmt.Errorf("telegram store: scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Delete removes the telegram target with the given id. Removing a missing
// target is a no-op.
func (r *TelegramStore) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM telegram_targets WHERE id = ?`, id); err != nil {
		return fmt.Errorf("telegram store: delete: %w", err)
	}
	return nil
}

// SetDefault marks id as the sole default, clearing any previous default in the
// same transaction so the single-default index is never violated.
func (r *TelegramStore) SetDefault(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("telegram store: set default: begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE telegram_targets SET is_default = 0 WHERE is_default = 1`); err != nil {
		return fmt.Errorf("telegram store: clear default: %w", err)
	}

	res, err := tx.ExecContext(ctx, `UPDATE telegram_targets SET is_default = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("telegram store: mark default: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return telegram.ErrNotFound
	}
	return tx.Commit()
}

// GetDefault returns the default telegram target, or ErrNotFound if none is set.
func (r *TelegramStore) GetDefault(ctx context.Context) (telegram.Target, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+telegramCols+` FROM telegram_targets WHERE is_default = 1`)
	t, err := scanTelegram(row)
	if errors.Is(err, sql.ErrNoRows) {
		return telegram.Target{}, telegram.ErrNotFound
	}
	if err != nil {
		return telegram.Target{}, fmt.Errorf("telegram store: get default: %w", err)
	}
	return t, nil
}

// SetBot marks id as the sole interactive-bot target, clearing any previous bot
// in the same transaction so at most one target is ever the bot.
func (r *TelegramStore) SetBot(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("telegram store: set bot: begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE telegram_targets SET is_bot = 0 WHERE is_bot = 1`); err != nil {
		return fmt.Errorf("telegram store: clear bot: %w", err)
	}

	res, err := tx.ExecContext(ctx, `UPDATE telegram_targets SET is_bot = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("telegram store: mark bot: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return telegram.ErrNotFound
	}
	return tx.Commit()
}

// GetBot returns the interactive-bot telegram target, or ErrNotFound if none is
// set.
func (r *TelegramStore) GetBot(ctx context.Context) (telegram.Target, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+telegramCols+` FROM telegram_targets WHERE is_bot = 1`)
	t, err := scanTelegram(row)
	if errors.Is(err, sql.ErrNoRows) {
		return telegram.Target{}, telegram.ErrNotFound
	}
	if err != nil {
		return telegram.Target{}, fmt.Errorf("telegram store: get bot: %w", err)
	}
	return t, nil
}

func scanTelegram(s scanner) (telegram.Target, error) {
	var (
		t         telegram.Target
		isDefault int
		isBot     int
		createdAt string
	)
	if err := s.Scan(&t.ID, &t.Name, &t.ChatID, &t.ThreadID, &t.TokenRef, &isDefault, &isBot, &createdAt); err != nil {
		return telegram.Target{}, err
	}
	t.IsDefault = isDefault != 0
	t.IsBot = isBot != 0
	parsed, err := parseTime(createdAt)
	if err != nil {
		return telegram.Target{}, err
	}
	t.CreatedAt = parsed
	return t, nil
}
