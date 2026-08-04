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
	if got.WebhookEnabled || got.WebhookSecret != "" || got.WebhookRequireConfirmation {
		t.Fatalf("new repo webhook = (enabled=%v, secret=%q, requireConfirmation=%v), want (false, \"\", false)", got.WebhookEnabled, got.WebhookSecret, got.WebhookRequireConfirmation)
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

	if err := s.SetWebhook(ctx, rp.ID, true, "s3cr3t-token", true); err != nil {
		t.Fatalf("SetWebhook: %v", err)
	}
	got, err := s.Get(ctx, rp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.WebhookEnabled || got.WebhookSecret != "s3cr3t-token" || !got.WebhookRequireConfirmation {
		t.Fatalf("after enable: (enabled=%v, secret=%q, requireConfirmation=%v), want (true, \"s3cr3t-token\", true)", got.WebhookEnabled, got.WebhookSecret, got.WebhookRequireConfirmation)
	}

	// It must not disturb the other fields.
	if got.Name != "web" || got.URL != "u" || got.AccountID != accID {
		t.Fatalf("SetWebhook mutated other fields: %+v", got)
	}

	// Disabling keeps the secret and can clear the confirmation gate.
	if err := s.SetWebhook(ctx, rp.ID, false, "s3cr3t-token", false); err != nil {
		t.Fatalf("SetWebhook disable: %v", err)
	}
	got2, err := s.Get(ctx, rp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got2.WebhookEnabled || got2.WebhookSecret != "s3cr3t-token" || got2.WebhookRequireConfirmation {
		t.Fatalf("after disable: (enabled=%v, secret=%q, requireConfirmation=%v), want (false, kept secret, false)", got2.WebhookEnabled, got2.WebhookSecret, got2.WebhookRequireConfirmation)
	}
}

// TestRepoSetWebhookUnknown asserts SetWebhook on a missing repo is ErrNotFound.
func TestRepoSetWebhookUnknown(t *testing.T) {
	ctx := context.Background()
	s, _ := newRepoStore(t)
	if err := s.SetWebhook(ctx, "does-not-exist", true, "x", false); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("SetWebhook unknown = %v, want repo.ErrNotFound", err)
	}
}
