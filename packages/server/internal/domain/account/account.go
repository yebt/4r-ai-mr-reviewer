// Package account models a GitLab account used to reach repositories.
package account

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when an account does not exist.
var ErrNotFound = errors.New("account: not found")

// Account is a configured GitLab connection. The token is not stored here;
// TokenRef names the entry in the secret store.
type Account struct {
	ID        string
	Name      string
	BaseURL   string
	TokenRef  string
	CreatedAt time.Time
}

// Repository persists accounts (not their tokens).
type Repository interface {
	Create(ctx context.Context, a Account) error
	Get(ctx context.Context, id string) (Account, error)
	List(ctx context.Context) ([]Account, error)
	Delete(ctx context.Context, id string) error
}
