// Package providers coordinates AI provider persistence with the encrypted
// secret store and the default-provider rule.
package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// Service manages providers and their API keys together.
type Service struct {
	repo    provider.Repository
	secrets secret.Store
}

// NewService wires the provider service.
func NewService(repo provider.Repository, secrets secret.Store) *Service {
	return &Service{repo: repo, secrets: secrets}
}

// AddInput describes a new provider.
type AddInput struct {
	Name      string
	Kind      provider.Kind
	BaseURL   string
	Model     string
	APIKey    string
	MakeDefault bool
}

// Add encrypts the API key and records the provider. The first provider added
// becomes the default automatically.
func (s *Service) Add(ctx context.Context, in AddInput) (provider.Provider, error) {
	if in.Name == "" || in.APIKey == "" {
		return provider.Provider{}, fmt.Errorf("providers: name and apiKey are required")
	}
	if !in.Kind.Valid() {
		return provider.Provider{}, fmt.Errorf("providers: invalid kind %q", in.Kind)
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return provider.Provider{}, err
	}
	makeDefault := in.MakeDefault || len(existing) == 0

	p := provider.Provider{
		ID:        id.New(),
		Name:      in.Name,
		Kind:      in.Kind,
		BaseURL:   in.BaseURL,
		Model:     in.Model,
		CreatedAt: time.Now().UTC(),
	}
	p.APIKeyRef = "provider:" + p.ID + ":apikey"

	if err := s.secrets.Set(ctx, p.APIKeyRef, []byte(in.APIKey)); err != nil {
		return provider.Provider{}, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		_ = s.secrets.Delete(ctx, p.APIKeyRef)
		return provider.Provider{}, err
	}
	if makeDefault {
		if err := s.repo.SetDefault(ctx, p.ID); err != nil {
			return provider.Provider{}, err
		}
		p.IsDefault = true
	}
	return p, nil
}

// List returns all providers.
func (s *Service) List(ctx context.Context) ([]provider.Provider, error) {
	return s.repo.List(ctx)
}

// Get returns one provider.
func (s *Service) Get(ctx context.Context, id string) (provider.Provider, error) {
	return s.repo.Get(ctx, id)
}

// Default returns the default provider, or provider.ErrNotFound if none set.
func (s *Service) Default(ctx context.Context) (provider.Provider, error) {
	return s.repo.GetDefault(ctx)
}

// SetDefault makes id the default provider.
func (s *Service) SetDefault(ctx context.Context, id string) error {
	return s.repo.SetDefault(ctx, id)
}

// APIKey returns the decrypted API key for a provider.
func (s *Service) APIKey(ctx context.Context, id string) (string, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	b, err := s.secrets.Get(ctx, p.APIKeyRef)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Remove deletes the provider and its API key.
func (s *Service) Remove(ctx context.Context, id string) error {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.secrets.Delete(ctx, p.APIKeyRef)
}
