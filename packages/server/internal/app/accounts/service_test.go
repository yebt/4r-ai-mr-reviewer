package accounts

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
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
	return NewService(sqlite.NewAccountRepo(db), secrets)
}

func TestAddAndToken(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	a, err := s.Add(ctx, "work", "https://gitlab.com", "glpat-xyz")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if a.ID == "" || a.TokenRef == "" {
		t.Fatalf("Add returned incomplete account: %+v", a)
	}

	tok, err := s.Token(ctx, a.ID)
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok != "glpat-xyz" {
		t.Fatalf("Token = %q, want glpat-xyz", tok)
	}
}

func TestUpdateAccount(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	a, err := s.Add(ctx, "work", "https://gitlab.com", "glpat-old")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Edit name + base URL, empty token keeps the stored token.
	up, err := s.Update(ctx, a.ID, "work-renamed", "https://gitlab.example.org", "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if up.Name != "work-renamed" || up.BaseURL != "https://gitlab.example.org" {
		t.Fatalf("edit not applied: %+v", up)
	}
	if tok, _ := s.Token(ctx, a.ID); tok != "glpat-old" {
		t.Fatalf("empty token should keep stored token, got %q", tok)
	}

	// A non-empty token rotates it.
	if _, err := s.Update(ctx, a.ID, "work-renamed", "https://gitlab.example.org", "glpat-new"); err != nil {
		t.Fatalf("Update rotate: %v", err)
	}
	if tok, _ := s.Token(ctx, a.ID); tok != "glpat-new" {
		t.Fatalf("token not rotated, got %q", tok)
	}

	// Unknown account → ErrNotFound.
	if _, err := s.Update(ctx, "nope", "n", "https://gitlab.com", ""); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("unknown Update = %v, want ErrNotFound", err)
	}

	// Insecure base URL is rejected.
	if _, err := s.Update(ctx, a.ID, "n", "http://gitlab.com", ""); err == nil {
		t.Fatal("expected insecure base URL rejection")
	}
}

func TestAddValidation(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), "", "url", "tok"); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestAddRejectsPublicHTTPBaseURL(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), "work", "http://gitlab.com", "glpat-xyz"); err == nil {
		t.Fatal("expected rejection of cleartext http base URL to a public host")
	}
}

func TestRemoveDeletesToken(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	a, _ := s.Add(ctx, "work", "https://gitlab.com", "glpat-xyz")
	if err := s.Remove(ctx, a.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := s.Get(ctx, a.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("Get after remove: got %v, want ErrNotFound", err)
	}
	// The token must be gone from the secret store too.
	if _, err := s.secrets.Get(ctx, a.TokenRef); !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("token after remove: got %v, want secret.ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	_, _ = s.Add(ctx, "a", "https://gitlab.com", "t1")
	_, _ = s.Add(ctx, "b", "https://gitlab.com", "t2")

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List len = %d, want 2", len(got))
	}
}
