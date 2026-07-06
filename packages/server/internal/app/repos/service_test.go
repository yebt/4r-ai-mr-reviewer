package repos

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

type fixture struct {
	svc          *Service
	accountRepo  *sqlite.AccountRepo
	providerRepo *sqlite.ProviderRepo
	accountID    string
	providerID   string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ar := sqlite.NewAccountRepo(db)
	pr := sqlite.NewProviderRepo(db)
	rs := sqlite.NewRepoStore(db)

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "acc", BaseURL: "https://gitlab.com", TokenRef: "ref", CreatedAt: time.Now().UTC()}
	if err := ar.Create(ctx, acc); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	prov := provider.Provider{ID: id.New(), Name: "p", Kind: provider.KindOpenAICompat, APIKeyRef: "ref", CreatedAt: time.Now().UTC()}
	if err := pr.Create(ctx, prov); err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	return fixture{
		svc:          NewService(rs, ar, pr),
		accountRepo:  ar,
		providerRepo: pr,
		accountID:    acc.ID,
		providerID:   prov.ID,
	}
}

func TestAddValidRepo(t *testing.T) {
	f := newFixture(t)
	r, err := f.svc.Add(context.Background(), AddInput{
		Name: "web", URL: "https://gitlab.com/g/web", AccountID: f.accountID,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if r.ID == "" || r.ProviderID != "" {
		t.Fatalf("unexpected repo: %+v", r)
	}
}

func TestAddRejectsUnknownAccount(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Add(context.Background(), AddInput{Name: "web", URL: "u", AccountID: "nope"})
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("got %v, want account.ErrNotFound", err)
	}
}

func TestAddRejectsUnknownProvider(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Add(context.Background(), AddInput{
		Name: "web", URL: "u", AccountID: f.accountID, ProviderID: "nope",
	})
	if !errors.Is(err, provider.ErrNotFound) {
		t.Fatalf("got %v, want provider.ErrNotFound", err)
	}
}

func TestAddMissingFields(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Add(context.Background(), AddInput{Name: "", URL: "u", AccountID: f.accountID}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestAssignProviderAndClear(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	r, _ := f.svc.Add(ctx, AddInput{Name: "web", URL: "u", AccountID: f.accountID})

	assigned, err := f.svc.Assign(ctx, r.ID, f.providerID, "gpt-4")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if assigned.ProviderID != f.providerID || assigned.Model != "gpt-4" {
		t.Fatalf("assign not applied: %+v", assigned)
	}

	cleared, err := f.svc.Assign(ctx, r.ID, "", "")
	if err != nil {
		t.Fatalf("Assign clear: %v", err)
	}
	if cleared.ProviderID != "" {
		t.Fatalf("expected cleared provider, got %q", cleared.ProviderID)
	}
}

func TestDeletingAccountCascadesRepos(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	r, _ := f.svc.Add(ctx, AddInput{Name: "web", URL: "u", AccountID: f.accountID})

	if err := f.accountRepo.Delete(ctx, f.accountID); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if _, err := f.svc.Get(ctx, r.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("repo should cascade-delete with its account, got %v", err)
	}
}

func TestDeletingProviderClearsAssignment(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	r, _ := f.svc.Add(ctx, AddInput{Name: "web", URL: "u", AccountID: f.accountID, ProviderID: f.providerID})

	if err := f.providerRepo.Delete(ctx, f.providerID); err != nil {
		t.Fatalf("delete provider: %v", err)
	}
	got, err := f.svc.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ProviderID != "" {
		t.Fatalf("provider assignment should be cleared to \"\", got %q", got.ProviderID)
	}
}
