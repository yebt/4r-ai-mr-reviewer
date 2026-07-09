// Package provider models a configured AI provider used to run reviews.
package provider

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a provider does not exist.
var ErrNotFound = errors.New("provider: not found")

// Kind identifies the wire protocol used to talk to a provider.
type Kind string

const (
	// KindOpenAICompat covers Groq, OpenAI, Moonshot, Kimi and OpenRouter.
	KindOpenAICompat Kind = "openai-compat"
	// KindAnthropic is the Claude messages API.
	KindAnthropic Kind = "anthropic"
)

// Valid reports whether k is a supported kind.
func (k Kind) Valid() bool {
	return k == KindOpenAICompat || k == KindAnthropic
}

// Provider is a configured AI backend. The API key is not stored here;
// APIKeyRef names the entry in the secret store.
type Provider struct {
	ID        string
	Name      string
	Kind      Kind
	BaseURL   string // optional; empty uses the provider's default endpoint
	Model     string // default model for reviews
	APIKeyRef string
	IsDefault bool
	// Temperature is an optional generation setting. nil means "don't send it"
	// (some models reject any value other than their default).
	Temperature *float64
	// Models is a list of preset model names to pick from when configuring a repo.
	Models    []string
	CreatedAt time.Time
}

// Repository persists providers (not their API keys).
type Repository interface {
	Create(ctx context.Context, p Provider) error
	Get(ctx context.Context, id string) (Provider, error)
	List(ctx context.Context) ([]Provider, error)
	// Update changes the mutable fields (name, kind, base URL, model).
	Update(ctx context.Context, p Provider) error
	Delete(ctx context.Context, id string) error
	// SetDefault marks id as the sole default provider.
	SetDefault(ctx context.Context, id string) error
	// GetDefault returns the default provider or ErrNotFound if none is set.
	GetDefault(ctx context.Context) (Provider, error)
}
