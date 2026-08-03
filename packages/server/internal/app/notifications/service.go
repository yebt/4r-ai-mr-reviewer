// Package notifications routes fired events to assigned notifier targets. A
// rule binds an event to a notifier target (currently Telegram) and can be
// toggled on or off. This replaces the previous hardcoded "notify the default
// Telegram target on review finish": the reviews service now calls Notify with
// an event, and this service fans out to every enabled rule for that event.
package notifications

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// ErrNotifierNotFound is returned when a rule is created against a notifier
// target that does not exist, so the rule can never silently no-op forever.
var ErrNotifierNotFound = errors.New("notifications: notifier target not found")

// ErrDuplicateRule is returned when a rule already exists for the same event
// and notifier target, so a fired event never delivers twice.
var ErrDuplicateRule = errors.New("notifications: rule already exists")

// Sender delivers text to a single notifier target and reports whether a target
// exists. It is implemented by *telegram.Service.
type Sender interface {
	SendTo(ctx context.Context, targetID, text string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// Service manages notification rules and fans out fired events to their
// assigned notifier targets.
type Service struct {
	rules    notification.Repository
	telegram Sender
}

// NewService wires the notifications service.
func NewService(rules notification.Repository, telegram Sender) *Service {
	return &Service{rules: rules, telegram: telegram}
}

// Events returns every event a rule may subscribe to.
func (s *Service) Events() []string {
	return notification.Events
}

// AddRule creates an enabled rule binding event to a notifier target. It
// validates the event and notifier kind and requires a non-empty target id.
//
// repoID optionally scopes the rule to a single repo. An empty repoID creates a
// GLOBAL rule. Callers that need to reject an unknown repoID must validate it
// before calling (the HTTP layer does this); this service does not know about
// repos.
func (s *Service) AddRule(ctx context.Context, event, notifierKind, notifierID, repoID string) (notification.Rule, error) {
	if !notification.ValidEvent(event) {
		return notification.Rule{}, fmt.Errorf("notifications: unknown event %q", event)
	}
	if notifierKind != notification.NotifierTelegram {
		return notification.Rule{}, fmt.Errorf("notifications: unsupported notifier kind %q", notifierKind)
	}
	if notifierID == "" {
		return notification.Rule{}, fmt.Errorf("notifications: notifierId is required")
	}
	// Reject a rule pointing at a nonexistent target: it would silently no-op on
	// every fired event forever.
	if notifierKind == notification.NotifierTelegram {
		exists, err := s.telegram.Exists(ctx, notifierID)
		if err != nil {
			return notification.Rule{}, err
		}
		if !exists {
			return notification.Rule{}, ErrNotifierNotFound
		}
	}
	// Reject a duplicate (same event + scope + target): duplicates would deliver
	// twice. Uniqueness is per (event, repoId, notifierId) so a global rule and a
	// repo-scoped rule for the same event and target can coexist. The DB UNIQUE
	// constraint is the safety net; this pre-check gives a clean error without
	// parsing driver-specific constraint messages.
	existing, err := s.rules.List(ctx)
	if err != nil {
		return notification.Rule{}, err
	}
	for _, r := range existing {
		if r.Event == event && r.RepoID == repoID && r.NotifierKind == notifierKind && r.NotifierID == notifierID {
			return notification.Rule{}, ErrDuplicateRule
		}
	}
	rule := notification.Rule{
		ID:           id.New(),
		Event:        event,
		NotifierKind: notifierKind,
		NotifierID:   notifierID,
		RepoID:       repoID,
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.rules.Create(ctx, rule); err != nil {
		return notification.Rule{}, err
	}
	return rule, nil
}

// ListRules returns all notification rules.
func (s *Service) ListRules(ctx context.Context) ([]notification.Rule, error) {
	return s.rules.List(ctx)
}

// GetRule returns one notification rule.
func (s *Service) GetRule(ctx context.Context, id string) (notification.Rule, error) {
	return s.rules.Get(ctx, id)
}

// SetRuleEnabled toggles a rule on or off.
func (s *Service) SetRuleEnabled(ctx context.Context, id string, enabled bool) error {
	return s.rules.SetEnabled(ctx, id, enabled)
}

// RemoveRule deletes a notification rule.
func (s *Service) RemoveRule(ctx context.Context, id string) error {
	return s.rules.Delete(ctx, id)
}

// RemoveRulesForNotifier deletes every rule targeting a notifier target, so
// deleting that target leaves no orphaned rules behind.
func (s *Service) RemoveRulesForNotifier(ctx context.Context, kind, id string) error {
	return s.rules.DeleteByNotifier(ctx, kind, id)
}

// Notify fans out an event fired for repoID to its enabled rules, applying
// OVERRIDE semantics: if any enabled rule is scoped to repoID, ONLY those
// repo-scoped rules fire; otherwise the global (unscoped) rules fire as a
// fallback. An empty repoID has no repo-scoped rules, so it always resolves to
// the global set.
//
// It is best-effort: a single failed delivery (or a missing target) is logged
// and skipped, and the call always returns nil so it never fails the caller (a
// finished review must never be held up by a notification). It satisfies the
// reviews.Notifier interface.
func (s *Service) Notify(ctx context.Context, event, repoID, text string) error {
	rules, err := s.rules.ListEnabledByEventForRepo(ctx, event, repoID)
	if err != nil {
		log.Printf("notifications: list rules for %q: %v", event, err)
		return nil
	}
	// Partition into the repo's own rules and the global fallback. When repoID is
	// empty, no rule matches the scoped set and every rule is global.
	var scoped, global []notification.Rule
	for _, rule := range rules {
		if repoID != "" && rule.RepoID == repoID {
			scoped = append(scoped, rule)
		} else if rule.RepoID == "" {
			global = append(global, rule)
		}
	}
	// OVERRIDE: a repo's own rules replace the global ones for that repo.
	targets := global
	if len(scoped) > 0 {
		targets = scoped
	}
	for _, rule := range targets {
		if rule.NotifierKind != notification.NotifierTelegram {
			continue
		}
		if err := s.telegram.SendTo(ctx, rule.NotifierID, text); err != nil {
			log.Printf("notifications: deliver rule %s: %v", rule.ID, err)
		}
	}
	return nil
}
