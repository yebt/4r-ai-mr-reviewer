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
}

// Repository persists tracked repos.
type Repository interface {
	Create(ctx context.Context, r Repo) error
	Get(ctx context.Context, id string) (Repo, error)
	List(ctx context.Context) ([]Repo, error)
	Update(ctx context.Context, r Repo) error
	Delete(ctx context.Context, id string) error
}
