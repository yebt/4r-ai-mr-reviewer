package notifications

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
)

// fakeSender records every SendTo call so tests can assert the fan-out without
// hitting the Bot API. missing lists target ids that Exists should report as
// absent; every other id is treated as existing.
type fakeSender struct {
	mu      sync.Mutex
	sent    []sentMessage
	err     error           // when set, every SendTo returns it
	missing map[string]bool // target ids Exists reports as absent
}

type sentMessage struct {
	targetID string
	text     string
}

func (f *fakeSender) SendTo(_ context.Context, targetID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{targetID: targetID, text: text})
	return f.err
}

func (f *fakeSender) Exists(_ context.Context, id string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.missing[id], nil
}

func (f *fakeSender) targets() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.sent))
	for _, m := range f.sent {
		out = append(out, m.targetID)
	}
	sort.Strings(out)
	return out
}

func newService(t *testing.T, sender Sender) *Service {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(sqlite.NewNotificationRuleStore(db), sender)
}

func TestAddRuleValidates(t *testing.T) {
	ctx := context.Background()
	s := newService(t, &fakeSender{})

	if _, err := s.AddRule(ctx, "bogus.event", notification.NotifierTelegram, "tg-1"); err == nil {
		t.Fatal("expected error for unknown event")
	}
	if _, err := s.AddRule(ctx, notification.EventReviewFinished, "slack", "tg-1"); err == nil {
		t.Fatal("expected error for unsupported notifier kind")
	}
	if _, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, ""); err == nil {
		t.Fatal("expected error for empty notifierId")
	}

	rule, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1")
	if err != nil {
		t.Fatalf("AddRule: %v", err)
	}
	if rule.ID == "" || !rule.Enabled {
		t.Fatalf("unexpected rule: %+v", rule)
	}
}

func TestAddRuleRejectsMissingTarget(t *testing.T) {
	ctx := context.Background()
	s := newService(t, &fakeSender{missing: map[string]bool{"ghost": true}})

	if _, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "ghost"); !errors.Is(err, ErrNotifierNotFound) {
		t.Fatalf("AddRule with missing target = %v, want ErrNotifierNotFound", err)
	}
}

func TestAddRuleRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	s := newService(t, &fakeSender{})

	if _, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1"); err != nil {
		t.Fatalf("first AddRule: %v", err)
	}
	if _, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1"); !errors.Is(err, ErrDuplicateRule) {
		t.Fatalf("duplicate AddRule = %v, want ErrDuplicateRule", err)
	}
}

func TestNotifyFansOutToEnabledRules(t *testing.T) {
	ctx := context.Background()
	sender := &fakeSender{}
	s := newService(t, sender)

	// Two enabled rules for the event, one disabled, one for another event.
	first, _ := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1")
	second, _ := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-2")
	disabled, _ := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-3")
	if err := s.SetRuleEnabled(ctx, disabled.ID, false); err != nil {
		t.Fatalf("SetRuleEnabled: %v", err)
	}

	if err := s.Notify(ctx, notification.EventReviewFinished, "review done"); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	got := sender.targets()
	want := []string{first.NotifierID, second.NotifierID}
	sort.Strings(want)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("fan-out targets = %v, want %v (disabled rule must be skipped)", got, want)
	}
}

func TestNotifyIsBestEffort(t *testing.T) {
	ctx := context.Background()
	sender := &fakeSender{err: context.DeadlineExceeded}
	s := newService(t, sender)

	if _, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1"); err != nil {
		t.Fatalf("AddRule: %v", err)
	}

	// A failing sender must never surface as a Notify error.
	if err := s.Notify(ctx, notification.EventReviewFinished, "boom"); err != nil {
		t.Fatalf("Notify must be best-effort, got %v", err)
	}
}

func TestSetRuleEnabledDisables(t *testing.T) {
	ctx := context.Background()
	sender := &fakeSender{}
	s := newService(t, sender)

	rule, _ := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1")
	if err := s.SetRuleEnabled(ctx, rule.ID, false); err != nil {
		t.Fatalf("SetRuleEnabled: %v", err)
	}

	if err := s.Notify(ctx, notification.EventReviewFinished, "review done"); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if len(sender.targets()) != 0 {
		t.Fatalf("disabled rule should not deliver, got %v", sender.targets())
	}
}

func TestRemoveRulesForNotifier(t *testing.T) {
	ctx := context.Background()
	s := newService(t, &fakeSender{})

	if _, err := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-1"); err != nil {
		t.Fatalf("AddRule: %v", err)
	}
	keep, _ := s.AddRule(ctx, notification.EventReviewFinished, notification.NotifierTelegram, "tg-2")

	if err := s.RemoveRulesForNotifier(ctx, notification.NotifierTelegram, "tg-1"); err != nil {
		t.Fatalf("RemoveRulesForNotifier: %v", err)
	}

	list, err := s.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(list) != 1 || list[0].ID != keep.ID {
		t.Fatalf("after removal: got %+v, want only %s", list, keep.ID)
	}
}
