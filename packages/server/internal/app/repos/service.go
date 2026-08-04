// Package repos manages tracked repositories, validating that the referenced
// account (and provider, when set) actually exist before persisting.
package repos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// Service manages tracked repos.
type Service struct {
	repos     repo.Repository
	accounts  account.Repository
	providers provider.Repository
}

// NewService wires the repo service.
func NewService(repos repo.Repository, accounts account.Repository, providers provider.Repository) *Service {
	return &Service{repos: repos, accounts: accounts, providers: providers}
}

// AddInput describes a new tracked repo.
type AddInput struct {
	Name       string
	URL        string
	AccountID  string
	ProviderID string // optional; "" means use the default provider
	Model      string // optional
}

// Add validates references and records the repo.
func (s *Service) Add(ctx context.Context, in AddInput) (repo.Repo, error) {
	if in.Name == "" || in.URL == "" || in.AccountID == "" {
		return repo.Repo{}, fmt.Errorf("repos: name, url and accountID are required")
	}
	if err := s.assertRefs(ctx, in.AccountID, in.ProviderID); err != nil {
		return repo.Repo{}, err
	}

	x := repo.Repo{
		ID:         id.New(),
		Name:       in.Name,
		URL:        in.URL,
		AccountID:  in.AccountID,
		ProviderID: in.ProviderID,
		Model:      in.Model,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repos.Create(ctx, x); err != nil {
		return repo.Repo{}, err
	}
	return x, nil
}

// List returns all repos.
func (s *Service) List(ctx context.Context) ([]repo.Repo, error) {
	return s.repos.List(ctx)
}

// Get returns one repo.
func (s *Service) Get(ctx context.Context, id string) (repo.Repo, error) {
	return s.repos.Get(ctx, id)
}

// Assign changes the provider and model of a repo. An empty providerID clears
// the assignment so the repo uses the default provider.
func (s *Service) Assign(ctx context.Context, id, providerID, model string) (repo.Repo, error) {
	x, err := s.repos.Get(ctx, id)
	if err != nil {
		return repo.Repo{}, err
	}
	if providerID != "" {
		if _, err := s.providers.Get(ctx, providerID); err != nil {
			return repo.Repo{}, err
		}
	}
	x.ProviderID = providerID
	x.Model = model
	if err := s.repos.Update(ctx, x); err != nil {
		return repo.Repo{}, err
	}
	return x, nil
}

// SetWebhook enables or disables the per-repo GitLab webhook. Enabling a repo
// that has no secret yet generates a strong random one; the secret is kept when
// disabling so re-enabling reuses it (the URL the user pasted into GitLab keeps
// working). Returns ErrNotFound when the repo does not exist.
func (s *Service) SetWebhook(ctx context.Context, id string, enabled bool) (repo.Repo, error) {
	x, err := s.repos.Get(ctx, id)
	if err != nil {
		return repo.Repo{}, err
	}
	secret := x.WebhookSecret
	if enabled && secret == "" {
		secret = newWebhookSecret()
	}
	if err := s.repos.SetWebhook(ctx, id, enabled, secret); err != nil {
		return repo.Repo{}, err
	}
	x.WebhookEnabled = enabled
	x.WebhookSecret = secret
	return x, nil
}

// RotateWebhookSecret replaces the repo's webhook secret with a fresh random one
// and returns the updated repo. The enabled flag is unchanged. Use it when the
// current token may be compromised; the user must then update the token in
// GitLab. Returns ErrNotFound when the repo does not exist.
func (s *Service) RotateWebhookSecret(ctx context.Context, id string) (repo.Repo, error) {
	x, err := s.repos.Get(ctx, id)
	if err != nil {
		return repo.Repo{}, err
	}
	secret := newWebhookSecret()
	if err := s.repos.SetWebhook(ctx, id, x.WebhookEnabled, secret); err != nil {
		return repo.Repo{}, err
	}
	x.WebhookSecret = secret
	return x, nil
}

// newWebhookSecret returns a strong random token (32 random bytes, hex-encoded)
// used as the per-repo GitLab webhook secret token.
func newWebhookSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read never fails on supported platforms; if it does, there
		// is no safe way to continue generating a secret.
		panic("repos: entropy source failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Remove deletes a repo.
func (s *Service) Remove(ctx context.Context, id string) error {
	return s.repos.Delete(ctx, id)
}

func (s *Service) assertRefs(ctx context.Context, accountID, providerID string) error {
	if _, err := s.accounts.Get(ctx, accountID); err != nil {
		return err
	}
	if providerID != "" {
		if _, err := s.providers.Get(ctx, providerID); err != nil {
			return err
		}
	}
	return nil
}
