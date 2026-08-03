package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
)

func newNotificationStore(t *testing.T) *NotificationRuleStore {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewNotificationRuleStore(db)
}

func mustCreateRule(t *testing.T, s *NotificationRuleStore, id, event, notifierID string, enabled bool) notification.Rule {
	t.Helper()
	return mustCreateScopedRule(t, s, id, event, notifierID, "", enabled)
}

func mustCreateScopedRule(t *testing.T, s *NotificationRuleStore, id, event, notifierID, repoID string, enabled bool) notification.Rule {
	t.Helper()
	rule := notification.Rule{
		ID:           id,
		Event:        event,
		NotifierKind: notification.NotifierTelegram,
		NotifierID:   notifierID,
		RepoID:       repoID,
		Enabled:      enabled,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
	}
	if err := s.Create(context.Background(), rule); err != nil {
		t.Fatalf("Create(%s): %v", id, err)
	}
	return rule
}

func TestNotificationStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newNotificationStore(t)

	want := notification.Rule{
		ID:           "r1",
		Event:        notification.EventReviewFinished,
		NotifierKind: notification.NotifierTelegram,
		NotifierID:   "tg-1",
		RepoID:       "repo-1",
		Enabled:      true,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
	}
	if err := s.Create(ctx, want); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, "r1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Event != want.Event || got.NotifierKind != want.NotifierKind || got.NotifierID != want.NotifierID || got.RepoID != want.RepoID || got.Enabled != want.Enabled {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("createdAt mismatch: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1", len(list))
	}
}

func TestNotificationStoreListEnabledByEvent(t *testing.T) {
	ctx := context.Background()
	s := newNotificationStore(t)

	enabled := mustCreateRule(t, s, "enabled", notification.EventReviewFinished, "tg-1", true)
	mustCreateRule(t, s, "disabled", notification.EventReviewFinished, "tg-2", false)
	mustCreateRule(t, s, "other-event", "other.event", "tg-3", true)

	got, err := s.ListEnabledByEvent(ctx, notification.EventReviewFinished)
	if err != nil {
		t.Fatalf("ListEnabledByEvent: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rules, want 1 (disabled and other-event must be filtered out)", len(got))
	}
	if got[0].ID != enabled.ID {
		t.Fatalf("got rule %s, want %s", got[0].ID, enabled.ID)
	}
}

func TestNotificationStoreListEnabledByEventForRepo(t *testing.T) {
	ctx := context.Background()
	s := newNotificationStore(t)

	globalRule := mustCreateScopedRule(t, s, "global", notification.EventReviewFinished, "tg-global", "", true)
	repoX := mustCreateScopedRule(t, s, "repo-x", notification.EventReviewFinished, "tg-x", "X", true)
	mustCreateScopedRule(t, s, "repo-y", notification.EventReviewFinished, "tg-y", "Y", true)
	// A disabled repo-X rule and an other-event global rule must be filtered out.
	mustCreateScopedRule(t, s, "repo-x-off", notification.EventReviewFinished, "tg-x2", "X", false)
	mustCreateScopedRule(t, s, "other", "other.event", "tg-z", "", true)

	got, err := s.ListEnabledByEventForRepo(ctx, notification.EventReviewFinished, "X")
	if err != nil {
		t.Fatalf("ListEnabledByEventForRepo: %v", err)
	}
	// Expect exactly the global rule and repo-X's enabled rule; repo Y and the
	// disabled/other-event rows are excluded.
	ids := map[string]bool{}
	for _, r := range got {
		ids[r.ID] = true
	}
	if len(got) != 2 || !ids[globalRule.ID] || !ids[repoX.ID] {
		t.Fatalf("got %d rules %v, want global %s + repo-X %s only", len(got), ids, globalRule.ID, repoX.ID)
	}
}

func TestNotificationStoreScopedRuleCoexistsWithGlobal(t *testing.T) {
	ctx := context.Background()
	s := newNotificationStore(t)

	// A global rule and a repo-scoped rule for the SAME event+notifier target must
	// both persist: uniqueness is (event, repo_id, notifier_id).
	mustCreateScopedRule(t, s, "g", notification.EventReviewFinished, "tg-1", "", true)
	mustCreateScopedRule(t, s, "s", notification.EventReviewFinished, "tg-1", "X", true)

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2 (global + repo-scoped coexist)", len(list))
	}
}

func TestNotificationStoreSetEnabled(t *testing.T) {
	ctx := context.Background()
	s := newNotificationStore(t)
	rule := mustCreateRule(t, s, "r1", notification.EventReviewFinished, "tg-1", true)

	if err := s.SetEnabled(ctx, rule.ID, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	got, _ := s.Get(ctx, rule.ID)
	if got.Enabled {
		t.Fatal("rule should be disabled after SetEnabled(false)")
	}

	if err := s.SetEnabled(ctx, "nope", true); !errors.Is(err, notification.ErrNotFound) {
		t.Fatalf("SetEnabled(unknown) = %v, want ErrNotFound", err)
	}
}

func TestNotificationStoreDeleteByNotifier(t *testing.T) {
	ctx := context.Background()
	s := newNotificationStore(t)

	// Two rules for the same notifier target on different events (the
	// (event, kind, id) UNIQUE constraint forbids two rows for the same event).
	mustCreateRule(t, s, "r1", notification.EventReviewFinished, "tg-1", true)
	mustCreateRule(t, s, "r2", "other.event", "tg-1", false)
	keep := mustCreateRule(t, s, "r3", notification.EventReviewFinished, "tg-2", true)

	if err := s.DeleteByNotifier(ctx, notification.NotifierTelegram, "tg-1"); err != nil {
		t.Fatalf("DeleteByNotifier: %v", err)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != keep.ID {
		t.Fatalf("after DeleteByNotifier: got %+v, want only %s", list, keep.ID)
	}
}

func TestNotificationStoreGetNotFound(t *testing.T) {
	s := newNotificationStore(t)
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, notification.ErrNotFound) {
		t.Fatalf("Get(unknown) = %v, want ErrNotFound", err)
	}
}
