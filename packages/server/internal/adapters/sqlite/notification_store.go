package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
)

// NotificationRuleStore is the SQLite-backed implementation of
// notification.Repository.
type NotificationRuleStore struct {
	db *sql.DB
}

// NewNotificationRuleStore wires a NotificationRuleStore over a database.
func NewNotificationRuleStore(db *sql.DB) *NotificationRuleStore {
	return &NotificationRuleStore{db: db}
}

var _ notification.Repository = (*NotificationRuleStore)(nil)

const notificationCols = `id, event, notifier_kind, notifier_id, enabled, created_at`

// Create inserts a notification rule row.
func (r *NotificationRuleStore) Create(ctx context.Context, rule notification.Rule) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification_rules(`+notificationCols+`) VALUES(?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.Event, rule.NotifierKind, rule.NotifierID, boolToInt(rule.Enabled), formatTime(rule.CreatedAt))
	if err != nil {
		return fmt.Errorf("notification store: create: %w", err)
	}
	return nil
}

// Get returns the notification rule with the given id.
func (r *NotificationRuleStore) Get(ctx context.Context, id string) (notification.Rule, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+notificationCols+` FROM notification_rules WHERE id = ?`, id)
	rule, err := scanNotificationRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return notification.Rule{}, notification.ErrNotFound
	}
	if err != nil {
		return notification.Rule{}, fmt.Errorf("notification store: get: %w", err)
	}
	return rule, nil
}

// List returns all notification rules, newest first.
func (r *NotificationRuleStore) List(ctx context.Context) ([]notification.Rule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+notificationCols+` FROM notification_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("notification store: list: %w", err)
	}
	defer rows.Close()

	var out []notification.Rule
	for rows.Next() {
		rule, err := scanNotificationRule(rows)
		if err != nil {
			return nil, fmt.Errorf("notification store: scan: %w", err)
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

// ListEnabledByEvent returns only the enabled rules subscribed to event.
func (r *NotificationRuleStore) ListEnabledByEvent(ctx context.Context, event string) ([]notification.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+notificationCols+` FROM notification_rules WHERE event = ? AND enabled = 1 ORDER BY created_at DESC`, event)
	if err != nil {
		return nil, fmt.Errorf("notification store: list enabled by event: %w", err)
	}
	defer rows.Close()

	var out []notification.Rule
	for rows.Next() {
		rule, err := scanNotificationRule(rows)
		if err != nil {
			return nil, fmt.Errorf("notification store: scan: %w", err)
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

// SetEnabled toggles a rule on or off. Toggling a missing rule returns
// ErrNotFound.
func (r *NotificationRuleStore) SetEnabled(ctx context.Context, id string, enabled bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE notification_rules SET enabled = ? WHERE id = ?`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("notification store: set enabled: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notification.ErrNotFound
	}
	return nil
}

// Delete removes the notification rule with the given id. Removing a missing
// rule is a no-op.
func (r *NotificationRuleStore) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM notification_rules WHERE id = ?`, id); err != nil {
		return fmt.Errorf("notification store: delete: %w", err)
	}
	return nil
}

// DeleteByNotifier removes every rule targeting a given notifier target.
func (r *NotificationRuleStore) DeleteByNotifier(ctx context.Context, kind, id string) error {
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM notification_rules WHERE notifier_kind = ? AND notifier_id = ?`, kind, id); err != nil {
		return fmt.Errorf("notification store: delete by notifier: %w", err)
	}
	return nil
}

func scanNotificationRule(s scanner) (notification.Rule, error) {
	var (
		rule      notification.Rule
		enabled   int
		createdAt string
	)
	if err := s.Scan(&rule.ID, &rule.Event, &rule.NotifierKind, &rule.NotifierID, &enabled, &createdAt); err != nil {
		return notification.Rule{}, err
	}
	rule.Enabled = enabled != 0
	parsed, err := parseTime(createdAt)
	if err != nil {
		return notification.Rule{}, err
	}
	rule.CreatedAt = parsed
	return rule, nil
}
