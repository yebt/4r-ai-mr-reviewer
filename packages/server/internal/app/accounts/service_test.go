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

func TestAddValidation(t *testing.T) {
	s := newService(t)
	if _, err := s.Add(context.Background(), "", "url", "tok"); err == nil {
		t.Fatal("expected validation error for empty name")
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
