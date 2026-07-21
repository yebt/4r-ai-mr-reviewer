// Package notification models assignable notification rules: an event fires a
// notification that is routed to an assigned notifier target (currently
// Telegram). Each rule can be toggled on or off.
package notification

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a notification rule does not exist.
var ErrNotFound = errors.New("notification: rule not found")

// EventReviewFinished fires when a review reaches a terminal state (done or
// error).
const EventReviewFinished = "review.finished"

// Events lists every event a rule may subscribe to.
var Events = []string{EventReviewFinished}

// ValidEvent reports whether e is a known event.
func ValidEvent(e string) bool {
	for _, ev := range Events {
		if ev == e {
			return true
		}
	}
	return false
}

// NotifierTelegram is the only notifier kind supported today.
const NotifierTelegram = "telegram"

// Rule routes an event to a notifier target. NotifierKind selects the notifier
// backend (e.g. telegram) and NotifierID identifies the concrete target within
// that backend (e.g. a telegram target id).
type Rule struct {
	ID           string
	Event        string
	NotifierKind string
	NotifierID   string
	Enabled      bool
	CreatedAt    time.Time
}

// Repository persists notification rules.
type Repository interface {
	Create(ctx context.Context, r Rule) error
	Get(ctx context.Context, id string) (Rule, error)
	List(ctx context.Context) ([]Rule, error)
	// ListEnabledByEvent returns only the enabled rules subscribed to event.
	ListEnabledByEvent(ctx context.Context, event string) ([]Rule, error)
	SetEnabled(ctx context.Context, id string, enabled bool) error
	Delete(ctx context.Context, id string) error
	// DeleteByNotifier removes every rule targeting a given notifier target, so
	// deleting that target leaves no orphaned rules behind.
	DeleteByNotifier(ctx context.Context, kind, id string) error
}
