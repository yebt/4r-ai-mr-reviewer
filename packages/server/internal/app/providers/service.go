// Package providers coordinates AI provider persistence with the encrypted
// secret store and the default-provider rule.
package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/ai"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
	"github.com/webcloster-dev/ai-reviewer/internal/netutil"
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
	Name        string
	Kind        provider.Kind
	BaseURL     string
	Model       string
	APIKey      string
	MakeDefault bool
	Temperature *float64
	Models      []string
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
	if err := netutil.RequireSecureBaseURL(in.BaseURL); err != nil {
		return provider.Provider{}, fmt.Errorf("providers: %w", err)
	}

	existing, err := s.repo.List(ctx)
	if err != nil {
		return provider.Provider{}, err
	}
	makeDefault := in.MakeDefault || len(existing) == 0

	p := provider.Provider{
		ID:          id.New(),
		Name:        in.Name,
		Kind:        in.Kind,
		BaseURL:     in.BaseURL,
		Model:       in.Model,
		Temperature: in.Temperature,
		Models:      in.Models,
		CreatedAt:   time.Now().UTC(),
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

// UpdateInput describes an edit to a provider. APIKey is optional: empty leaves
// the stored key unchanged.
type UpdateInput struct {
	Name        string
	Kind        provider.Kind
	BaseURL     string
	Model       string
	APIKey      string
	Temperature *float64
	Models      []string
}

// Update edits a provider's fields, optionally rotating its API key.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (provider.Provider, error) {
	if in.Name == "" {
		return provider.Provider{}, fmt.Errorf("providers: name is required")
	}
	if !in.Kind.Valid() {
		return provider.Provider{}, fmt.Errorf("providers: invalid kind %q", in.Kind)
	}
	if err := netutil.RequireSecureBaseURL(in.BaseURL); err != nil {
		return provider.Provider{}, fmt.Errorf("providers: %w", err)
	}

	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return provider.Provider{}, err
	}
	p.Name = in.Name
	p.Kind = in.Kind
	p.BaseURL = in.BaseURL
	p.Model = in.Model
	p.Temperature = in.Temperature
	p.Models = in.Models

	if err := s.repo.Update(ctx, p); err != nil {
		return provider.Provider{}, err
	}
	if in.APIKey != "" {
		if err := s.secrets.Set(ctx, p.APIKeyRef, []byte(in.APIKey)); err != nil {
			return provider.Provider{}, err
		}
	}
	return p, nil
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

// TestInput describes a provider configuration to probe. When ID is set and
// APIKey is empty, the stored key for that provider is used, so an edit form can
// test the connection without re-entering the key.
type TestInput struct {
	ID      string
	Kind    provider.Kind
	BaseURL string
	Model   string
	APIKey  string
}

// Test performs one minimal live call against the provider to verify that the
// base URL, API key and model actually work — surfacing credential/endpoint/model
// errors at configuration time instead of inside a later long-running review. It
// persists nothing.
func (s *Service) Test(ctx context.Context, in TestInput) error {
	if !in.Kind.Valid() {
		return fmt.Errorf("providers: invalid kind %q", in.Kind)
	}
	if in.Model == "" {
		return fmt.Errorf("providers: a model is required to test the connection")
	}
	if err := netutil.RequireSecureBaseURL(in.BaseURL); err != nil {
		return fmt.Errorf("providers: %w", err)
	}

	key := in.APIKey
	if key == "" {
		if in.ID == "" {
			return fmt.Errorf("providers: an API key is required to test the connection")
		}
		var err error
		if key, err = s.APIKey(ctx, in.ID); err != nil {
			return err
		}
	}

	client, err := ai.New(provider.Provider{Kind: in.Kind, BaseURL: in.BaseURL, Model: in.Model}, key)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err = client.Complete(ctx, llm.Request{
		Model:     in.Model,
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "ping"}},
		MaxTokens: 16,
	})
	return err
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
