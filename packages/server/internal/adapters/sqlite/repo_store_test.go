package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// newRepoStore opens a fresh DB, seeds an account, and returns a RepoStore.
func newRepoStore(t *testing.T) (*RepoStore, string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "a", BaseURL: "https://gitlab.com", TokenRef: "ref", CreatedAt: time.Now().UTC()}
	if err := NewAccountRepo(db).Create(ctx, acc); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return NewRepoStore(db), acc.ID
}

// TestRepoWebhookDefaults asserts a freshly created repo starts with the webhook
// disabled and an empty secret.
func TestRepoWebhookDefaults(t *testing.T) {
	ctx := context.Background()
	s, accID := newRepoStore(t)

	rp := repo.Repo{ID: id.New(), Name: "web", URL: "u", AccountID: accID, CreatedAt: time.Now().UTC()}
	if err := s.Create(ctx, rp); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, rp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.WebhookEnabled || got.WebhookSecret != "" {
		t.Fatalf("new repo webhook = (enabled=%v, secret=%q), want (false, \"\")", got.WebhookEnabled, got.WebhookSecret)
	}
}

// TestRepoSetWebhookRoundTrip asserts SetWebhook persists the enable flag and
// secret, and scanRepo reads them back.
func TestRepoSetWebhookRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, accID := newRepoStore(t)

	rp := repo.Repo{ID: id.New(), Name: "web", URL: "u", AccountID: accID, CreatedAt: time.Now().UTC()}
	if err := s.Create(ctx, rp); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.SetWebhook(ctx, rp.ID, true, "s3cr3t-token"); err != nil {
		t.Fatalf("SetWebhook: %v", err)
	}
	got, err := s.Get(ctx, rp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.WebhookEnabled || got.WebhookSecret != "s3cr3t-token" {
		t.Fatalf("after enable: (enabled=%v, secret=%q), want (true, \"s3cr3t-token\")", got.WebhookEnabled, got.WebhookSecret)
	}

	// It must not disturb the other fields.
	if got.Name != "web" || got.URL != "u" || got.AccountID != accID {
		t.Fatalf("SetWebhook mutated other fields: %+v", got)
	}

	// Disabling keeps the secret.
	if err := s.SetWebhook(ctx, rp.ID, false, "s3cr3t-token"); err != nil {
		t.Fatalf("SetWebhook disable: %v", err)
	}
	got2, err := s.Get(ctx, rp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got2.WebhookEnabled || got2.WebhookSecret != "s3cr3t-token" {
		t.Fatalf("after disable: (enabled=%v, secret=%q), want (false, kept secret)", got2.WebhookEnabled, got2.WebhookSecret)
	}
}

// TestRepoSetWebhookUnknown asserts SetWebhook on a missing repo is ErrNotFound.
func TestRepoSetWebhookUnknown(t *testing.T) {
	ctx := context.Background()
	s, _ := newRepoStore(t)
	if err := s.SetWebhook(ctx, "does-not-exist", true, "x"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("SetWebhook unknown = %v, want repo.ErrNotFound", err)
	}
}
