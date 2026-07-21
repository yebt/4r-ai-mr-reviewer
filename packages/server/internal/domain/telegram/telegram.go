// Package telegram models a configured Telegram notification target. The bot
// token is not stored here; TokenRef names the encrypted entry in the secret
// store.
package telegram

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a telegram target does not exist.
var ErrNotFound = errors.New("telegram: not found")

// Target is a configured Telegram chat to notify. The bot token is not stored
// here; TokenRef names the entry in the secret store.
type Target struct {
	ID   string
	Name string
	// ChatID is the destination chat (user, group or channel).
	ChatID string
	// ThreadID optionally targets a forum topic; empty means the main chat.
	ThreadID  string
	TokenRef  string
	IsDefault bool
	CreatedAt time.Time
}

// Repository persists telegram targets (not their bot tokens).
type Repository interface {
	Create(ctx context.Context, t Target) error
	Get(ctx context.Context, id string) (Target, error)
	List(ctx context.Context) ([]Target, error)
	Delete(ctx context.Context, id string) error
	// SetDefault marks id as the sole default target.
	SetDefault(ctx context.Context, id string) error
	// GetDefault returns the default target, or ErrNotFound if none is set.
	GetDefault(ctx context.Context) (Target, error)
}
