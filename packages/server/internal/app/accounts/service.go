// Package accounts coordinates GitLab account persistence with the encrypted
// secret store, so a token is always stored and removed alongside its account.
package accounts

import (
	"context"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// Service manages accounts and their tokens together.
type Service struct {
	repo    account.Repository
	secrets secret.Store
}

// NewService wires the account service.
func NewService(repo account.Repository, secrets secret.Store) *Service {
	return &Service{repo: repo, secrets: secrets}
}

// Add encrypts the token and records the account.
func (s *Service) Add(ctx context.Context, name, baseURL, token string) (account.Account, error) {
	if name == "" || baseURL == "" || token == "" {
		return account.Account{}, fmt.Errorf("accounts: name, baseURL and token are required")
	}
	a := account.Account{
		ID:        id.New(),
		Name:      name,
		BaseURL:   baseURL,
		CreatedAt: time.Now().UTC(),
	}
	a.TokenRef = "account:" + a.ID + ":token"

	if err := s.secrets.Set(ctx, a.TokenRef, []byte(token)); err != nil {
		return account.Account{}, err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		// Roll back the orphaned secret so we never leak a dangling token.
		_ = s.secrets.Delete(ctx, a.TokenRef)
		return account.Account{}, err
	}
	return a, nil
}

// List returns all accounts.
func (s *Service) List(ctx context.Context) ([]account.Account, error) {
	return s.repo.List(ctx)
}

// Get returns one account.
func (s *Service) Get(ctx context.Context, id string) (account.Account, error) {
	return s.repo.Get(ctx, id)
}

// Token returns the decrypted token for an account.
func (s *Service) Token(ctx context.Context, id string) (string, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	b, err := s.secrets.Get(ctx, a.TokenRef)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Remove deletes the account and its token.
func (s *Service) Remove(ctx context.Context, id string) error {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.secrets.Delete(ctx, a.TokenRef)
}
