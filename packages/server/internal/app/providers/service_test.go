package providers

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
)

func newService(t *testing.T) *Service {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	return NewService(sqlite.NewProviderRepo(db), secrets)
}

func mustAdd(t *testing.T, s *Service, in AddInput) provider.Provider {
	t.Helper()
	p, err := s.Add(context.Background(), in)
	if err != nil {
		t.Fatalf("Add(%s): %v", in.Name, err)
	}
	return p
}

func TestFirstProviderBecomesDefault(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	p := mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, Model: "llama", APIKey: "k1"})
	if !p.IsDefault {
		t.Fatal("first provider should be default")
	}
	def, err := s.Default(ctx)
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if def.ID != p.ID {
		t.Fatalf("Default id = %s, want %s", def.ID, p.ID)
	}
}

func TestSecondProviderNotDefaultUnlessAsked(t *testing.T) {
	s := newService(t)
	_ = mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "k1"})
	second := mustAdd(t, s, AddInput{Name: "claude", Kind: provider.KindAnthropic, APIKey: "k2"})
	if second.IsDefault {
		t.Fatal("second provider should not become default automatically")
	}
}

func TestSetDefaultSwitches(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	first := mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "k1"})
	second := mustAdd(t, s, AddInput{Name: "claude", Kind: provider.KindAnthropic, APIKey: "k2"})

	if err := s.SetDefault(ctx, second.ID); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}
	def, _ := s.Default(ctx)
	if def.ID != second.ID {
		t.Fatalf("default id = %s, want %s (only one default allowed)", def.ID, second.ID)
	}
	// The former default must have been cleared.
	reloaded, _ := s.Get(ctx, first.ID)
	if reloaded.IsDefault {
		t.Fatal("former default should have been cleared")
	}
}

func TestAPIKeyRoundTripAndRemove(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "secret-key"})

	key, err := s.APIKey(ctx, p.ID)
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	if key != "secret-key" {
		t.Fatalf("APIKey = %q, want secret-key", key)
	}

	if err := s.Remove(ctx, p.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := s.secrets.Get(ctx, p.APIKeyRef); !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("api key after remove: got %v, want secret.ErrNotFound", err)
	}
}

func TestAddInvalidKind(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), AddInput{Name: "x", Kind: "bogus", APIKey: "k"}); err == nil {
		t.Fatal("expected invalid kind error")
	}
}

func TestAddRejectsPublicHTTPBaseURL(t *testing.T) {
	s := newService(t)
	_, err := s.Add(context.Background(), AddInput{
		Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "k",
		BaseURL: "http://api.groq.com/openai/v1",
	})
	if err == nil {
		t.Fatal("expected rejection of cleartext http base URL to a public host")
	}
}

func TestDefaultWhenNoneSet(t *testing.T) {
	s := newService(t)
	if _, err := s.Default(context.Background()); !errors.Is(err, provider.ErrNotFound) {
		t.Fatalf("Default with no providers: got %v, want ErrNotFound", err)
	}
}

func TestUpdateFieldsKeepsKeyWhenBlank(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, Model: "old", APIKey: "orig-key"})

	updated, err := s.Update(ctx, p.ID, UpdateInput{Name: "groq-eu", Kind: provider.KindOpenAICompat, Model: "new", BaseURL: "https://eu"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "groq-eu" || updated.Model != "new" || updated.BaseURL != "https://eu" {
		t.Fatalf("fields not updated: %+v", updated)
	}
	// Blank APIKey leaves the stored key intact.
	key, _ := s.APIKey(ctx, p.ID)
	if key != "orig-key" {
		t.Fatalf("API key changed on blank update: %q", key)
	}
}

func TestUpdateRotatesKeyWhenProvided(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "orig-key"})

	if _, err := s.Update(ctx, p.ID, UpdateInput{Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "rotated-key"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	key, _ := s.APIKey(ctx, p.ID)
	if key != "rotated-key" {
		t.Fatalf("API key not rotated: %q", key)
	}
}

func TestUpdateUnknownProvider(t *testing.T) {
	s := newService(t)
	if _, err := s.Update(context.Background(), "nope", UpdateInput{Name: "x", Kind: provider.KindOpenAICompat}); !errors.Is(err, provider.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestTemperatureAndModelsRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	temp := 0.2
	p := mustAdd(t, s, AddInput{
		Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "k",
		Temperature: &temp, Models: []string{"llama-3.3", "kimi-k2"},
	})

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Temperature == nil || *got.Temperature != 0.2 {
		t.Fatalf("temperature not persisted: %v", got.Temperature)
	}
	if len(got.Models) != 2 || got.Models[0] != "llama-3.3" {
		t.Fatalf("models not persisted: %+v", got.Models)
	}
}

func TestUpdateClearsTemperature(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	temp := 0.5
	p := mustAdd(t, s, AddInput{Name: "groq", Kind: provider.KindOpenAICompat, APIKey: "k", Temperature: &temp})

	if _, err := s.Update(ctx, p.ID, UpdateInput{Name: "groq", Kind: provider.KindOpenAICompat, Temperature: nil}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(ctx, p.ID)
	if got.Temperature != nil {
		t.Fatalf("temperature should be cleared, got %v", *got.Temperature)
	}
}
