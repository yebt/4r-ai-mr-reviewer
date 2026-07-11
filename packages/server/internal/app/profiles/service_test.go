package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
)

// aiStub serves a canned OpenAI-compatible completion carrying the given text.
func aiStub(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m",
			"choices": []map[string]any{{"message": map[string]any{"content": content}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
}

// newServiceWithAI wires a profiles.Service over a real providers.Service. When
// aiURL is non-empty a default OpenAI-compatible provider pointing at it is
// registered (the first provider becomes the default automatically).
func newServiceWithAI(t *testing.T, aiURL string) *Service {
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
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)

	if aiURL != "" {
		if _, err := providerSvc.Add(context.Background(), providers.AddInput{
			Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiURL, Model: "m", APIKey: "k",
		}); err != nil {
			t.Fatalf("add provider: %v", err)
		}
	}
	// Discard logger so async distillation failures don't pollute test output.
	// A no-op distill runner keeps the background goroutine (and its db writes)
	// from outliving the test — the sync Distill path is exercised directly.
	return NewService(sqlite.NewProfileStore(db), providerSvc, log.New(io.Discard, "", 0), WithDistillRunner(func(string) {}))
}

func newService(t *testing.T) *Service {
	t.Helper()
	return newServiceWithAI(t, "")
}

func mustAdd(t *testing.T, s *Service, in AddInput) profile.Profile {
	t.Helper()
	p, err := s.Add(context.Background(), in)
	if err != nil {
		t.Fatalf("Add(%s): %v", in.Name, err)
	}
	return p
}

func TestAddSetsIDAndTimestamps(t *testing.T) {
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "casual", Language: "es-AR", Formality: "casual", Emojis: true, Samples: []string{"che"}})

	if p.ID == "" {
		t.Fatal("Add should assign an id")
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not set: %+v", p)
	}
	if p.StyleGuide != "" {
		t.Fatalf("styleGuide should stay empty, got %q", p.StyleGuide)
	}
}

func TestAddRequiresName(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), AddInput{Name: ""}); err == nil {
		t.Fatal("expected name required error")
	}
}

func TestAddListGet(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "one", Samples: []string{"a", "b"}})

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != p.ID {
		t.Fatalf("List = %+v", list)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Samples) != 2 || got.Samples[0] != "a" {
		t.Fatalf("samples not persisted: %+v", got.Samples)
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "old", Language: "en", Emojis: false, Samples: []string{"x"}})

	updated, err := s.Update(ctx, p.ID, UpdateInput{
		Name: "new", Language: "es", Formality: "formal", Emojis: true, Samples: []string{"y", "z"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "new" || updated.Language != "es" || updated.Formality != "formal" || !updated.Emojis {
		t.Fatalf("fields not updated: %+v", updated)
	}
	if len(updated.Samples) != 2 || updated.Samples[1] != "z" {
		t.Fatalf("samples not updated: %+v", updated.Samples)
	}
	if !updated.UpdatedAt.After(p.UpdatedAt) && updated.UpdatedAt.Before(p.UpdatedAt) {
		t.Fatalf("updatedAt not advanced")
	}
}

func TestUpdateRequiresName(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "one"})
	if _, err := s.Update(ctx, p.ID, UpdateInput{Name: ""}); err == nil {
		t.Fatal("expected name required error")
	}
}

func TestUpdateUnknown(t *testing.T) {
	s := newService(t)
	if _, err := s.Update(context.Background(), "nope", UpdateInput{Name: "x"}); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "gone"})

	if err := s.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, p.ID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestAddWithSamplesSetsPendingImmediately(t *testing.T) {
	// A stub keeps the async goroutine from erroring, but Add must return
	// pending synchronously without waiting for it.
	srv := aiStub(t, "guide")
	defer srv.Close()
	s := newServiceWithAI(t, srv.URL)

	p := mustAdd(t, s, AddInput{Name: "voice", Samples: []string{"hello there"}})
	if p.StyleGuideStatus != profile.StyleStatusPending {
		t.Fatalf("status = %q, want pending", p.StyleGuideStatus)
	}
	if p.StyleGuide != "" {
		t.Fatalf("styleGuide should be empty on the pending response, got %q", p.StyleGuide)
	}
}

func TestAddWithoutSamplesSetsNone(t *testing.T) {
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "empty"})
	if p.StyleGuideStatus != profile.StyleStatusNone {
		t.Fatalf("status = %q, want none", p.StyleGuideStatus)
	}
}

func TestDistillSetsStyleGuide(t *testing.T) {
	ctx := context.Background()
	const guide = "Voice: warm and terse. Short sentences. No emojis."
	srv := aiStub(t, "  "+guide+"  ")
	defer srv.Close()
	s := newServiceWithAI(t, srv.URL)

	p := mustAdd(t, s, AddInput{Name: "voice", Language: "en", Samples: []string{"hi", "there"}})
	if err := s.Distill(ctx, p.ID); err != nil {
		t.Fatalf("Distill: %v", err)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StyleGuide != guide {
		t.Fatalf("styleGuide = %q, want %q (trimmed)", got.StyleGuide, guide)
	}
	if got.StyleGuideStatus != profile.StyleStatusReady {
		t.Fatalf("status = %q, want ready", got.StyleGuideStatus)
	}
	if got.StyleGuideError != "" {
		t.Fatalf("error should be cleared, got %q", got.StyleGuideError)
	}
}

func TestDistillErrorSetsErrorStatus(t *testing.T) {
	ctx := context.Background()
	// A server that always 500s makes the LLM call fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	s := newServiceWithAI(t, srv.URL)

	p := mustAdd(t, s, AddInput{Name: "voice", Samples: []string{"hi"}})
	if err := s.Distill(ctx, p.ID); err == nil {
		t.Fatal("expected Distill to fail")
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StyleGuideStatus != profile.StyleStatusError {
		t.Fatalf("status = %q, want error", got.StyleGuideStatus)
	}
	if got.StyleGuideError == "" {
		t.Fatal("expected an error message to be stored")
	}
	if got.StyleGuide != "" {
		t.Fatalf("styleGuide should be empty on error, got %q", got.StyleGuide)
	}
}

func TestDistillNoSamplesIsNoop(t *testing.T) {
	ctx := context.Background()
	// No provider configured: if Distill tried to call the LLM it would fail,
	// proving it short-circuits on empty samples.
	s := newService(t)
	p := mustAdd(t, s, AddInput{Name: "empty"})

	if err := s.Distill(ctx, p.ID); err != nil {
		t.Fatalf("Distill on empty samples should be a noop, got %v", err)
	}
	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StyleGuideStatus != profile.StyleStatusNone {
		t.Fatalf("status = %q, want none", got.StyleGuideStatus)
	}
}
