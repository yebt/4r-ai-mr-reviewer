// Package repo models a tracked repository (a GitLab project) that reviews
// run against.
package repo

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a repo does not exist.
var ErrNotFound = errors.New("repo: not found")

// Repo is a tracked repository. It is reached through an Account and reviewed
// with a Provider; empty ProviderID/Model fall back to the default provider
// and its default model.
type Repo struct {
	ID         string
	Name       string
	URL        string
	AccountID  string
	ProviderID string // "" means use the default provider
	Model      string // "" means use the provider's default model
	CreatedAt  time.Time
	// WebhookSecret gates the per-repo GitLab webhook (compared against the
	// X-Gitlab-Token header). Empty means no secret has been generated yet.
	WebhookSecret string
	// WebhookEnabled toggles the auto-review receiver on for this repo. When
	// false the endpoint is inert even if it receives events.
	WebhookEnabled bool
	// WebhookRequireConfirmation holds a webhook-triggered review for manual
	// approval instead of running it immediately: when true the review is created
	// in the awaiting_approval state and only runs once the user approves it in
	// the app. When false (the default) webhook reviews auto-run as before.
	WebhookRequireConfirmation bool
}

// Repository persists tracked repos.
type Repository interface {
	Create(ctx context.Context, r Repo) error
	Get(ctx context.Context, id string) (Repo, error)
	List(ctx context.Context) ([]Repo, error)
	Update(ctx context.Context, r Repo) error
	Delete(ctx context.Context, id string) error
	// SetWebhook updates only the webhook enable flag, secret and the
	// require-confirmation gate. Returns ErrNotFound if the repo does not exist.
	SetWebhook(ctx context.Context, id string, enabled bool, secret string, requireConfirmation bool) error
}
