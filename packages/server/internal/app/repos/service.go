// Package repos manages tracked repositories, validating that the referenced
// account (and provider, when set) actually exist before persisting.
package repos

import (
	"context"
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
