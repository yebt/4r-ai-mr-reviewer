// Package profiles coordinates humanization profile persistence and the async
// distillation of writing samples into a reusable style guide. Profiles hold no
// secrets, so (unlike providers) the service has no secret-store dependency; it
// borrows the providers service only to resolve the default LLM for distillation.
package profiles

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/ai"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/humanize"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// distillTimeout bounds a single background distillation call. The request ctx
// is cancelled when the HTTP response is sent, so distillation runs on its own
// timed context.Background instead.
const distillTimeout = 2 * time.Minute

// Service manages humanization profiles.
type Service struct {
	repo       profile.Repository
	providers  *providers.Service
	logger     *log.Logger
	runDistill func(profileID string)
}

// Option configures a Service.
type Option func(*Service)

// WithDistillRunner overrides how async distillation is dispatched. Production
// uses the default (a timed background goroutine); tests inject a synchronous or
// no-op runner so a goroutine never outlives the test and its database.
func WithDistillRunner(fn func(profileID string)) Option {
	return func(s *Service) { s.runDistill = fn }
}

// NewService wires the profile service. providers resolves the default LLM used
// for style-guide distillation; logger receives async distillation failures
// (defaults to log.Default() when nil).
func NewService(repo profile.Repository, providers *providers.Service, logger *log.Logger, opts ...Option) *Service {
	if logger == nil {
		logger = log.Default()
	}
	s := &Service{repo: repo, providers: providers, logger: logger}
	s.runDistill = s.triggerDistill
	for _, o := range opts {
		o(s)
	}
	return s
}

// AddInput describes a new profile. StyleGuide is server-managed and stays empty
// until distillation fills it, so it is not accepted here.
type AddInput struct {
	Name      string
	Language  string
	Formality string
	Emojis    bool
	Samples   []string
}

// Add records a new profile and, when it carries samples, kicks off async
// distillation (returning immediately with a "pending" status).
func (s *Service) Add(ctx context.Context, in AddInput) (profile.Profile, error) {
	if in.Name == "" {
		return profile.Profile{}, fmt.Errorf("profiles: name is required")
	}

	now := time.Now().UTC()
	p := profile.Profile{
		ID:        id.New(),
		Name:      in.Name,
		Language:  in.Language,
		Formality: in.Formality,
		Emojis:    in.Emojis,
		Samples:   in.Samples,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return profile.Profile{}, err
	}
	return s.scheduleDistill(ctx, p)
}

// List returns all profiles.
func (s *Service) List(ctx context.Context) ([]profile.Profile, error) {
	return s.repo.List(ctx)
}

// Get returns one profile.
func (s *Service) Get(ctx context.Context, id string) (profile.Profile, error) {
	return s.repo.Get(ctx, id)
}

// UpdateInput describes an edit to a profile. StyleGuide is server-managed and
// is never overwritten here.
type UpdateInput struct {
	Name      string
	Language  string
	Formality string
	Emojis    bool
	Samples   []string
}

// Update edits a profile's fields and re-runs distillation when it carries
// samples (or clears a stale guide when the samples were removed).
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (profile.Profile, error) {
	if in.Name == "" {
		return profile.Profile{}, fmt.Errorf("profiles: name is required")
	}

	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return profile.Profile{}, err
	}
	p.Name = in.Name
	p.Language = in.Language
	p.Formality = in.Formality
	p.Emojis = in.Emojis
	p.Samples = in.Samples
	p.UpdatedAt = time.Now().UTC()

	if err := s.repo.Save(ctx, p); err != nil {
		return profile.Profile{}, err
	}
	return s.scheduleDistill(ctx, p)
}

// Delete removes a profile.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// scheduleDistill sets the initial style-guide status for a freshly saved
// profile: pending (+ async trigger) when it has samples, none otherwise (which
// also clears any stale guide). It returns the profile with the new status set
// so the CRUD response reflects it immediately, without waiting for the LLM.
func (s *Service) scheduleDistill(ctx context.Context, p profile.Profile) (profile.Profile, error) {
	if len(p.Samples) == 0 {
		if err := s.repo.SetStyleGuide(ctx, p.ID, "", profile.StyleStatusNone, ""); err != nil {
			return profile.Profile{}, err
		}
		p.StyleGuide = ""
		p.StyleGuideStatus = profile.StyleStatusNone
		p.StyleGuideError = ""
		return p, nil
	}
	if err := s.repo.SetStyleGuide(ctx, p.ID, "", profile.StyleStatusPending, ""); err != nil {
		return profile.Profile{}, err
	}
	p.StyleGuide = ""
	p.StyleGuideStatus = profile.StyleStatusPending
	p.StyleGuideError = ""
	s.runDistill(p.ID)
	return p, nil
}

// Distill synchronously distills a profile's stored samples into its style guide
// via one LLM call on the default provider, and persists the outcome. It uses
// the profile's CURRENT stored samples/knobs (not any caller-supplied input).
// This is the unit-testable core; the async path wraps it.
func (s *Service) Distill(ctx context.Context, profileID string) error {
	p, err := s.repo.Get(ctx, profileID)
	if err != nil {
		return err
	}
	if len(p.Samples) == 0 {
		return s.repo.SetStyleGuide(ctx, profileID, "", profile.StyleStatusNone, "")
	}

	guide, err := s.distill(ctx, p)
	if err != nil {
		_ = s.repo.SetStyleGuide(ctx, profileID, "", profile.StyleStatusError, err.Error())
		return err
	}
	return s.repo.SetStyleGuide(ctx, profileID, guide, profile.StyleStatusReady, "")
}

// distill resolves the default provider, builds the AI client, and runs the
// single distillation completion, returning the trimmed style-guide text.
func (s *Service) distill(ctx context.Context, p profile.Profile) (string, error) {
	prov, err := s.providers.Default(ctx)
	if err != nil {
		return "", err
	}
	apiKey, err := s.providers.APIKey(ctx, prov.ID)
	if err != nil {
		return "", err
	}
	model := prov.Model
	if model == "" {
		return "", fmt.Errorf("profiles: no model set on default provider %q", prov.Name)
	}
	client, err := ai.New(prov, apiKey)
	if err != nil {
		return "", err
	}
	resp, err := client.Complete(ctx, llm.Request{
		Model:       model,
		Messages:    humanize.BuildDistillMessages(p),
		Temperature: prov.Temperature,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

// triggerDistill fires Distill on a fresh, timed background context. It is a
// thin wrapper with no logic of its own; failures are logged (the persisted
// error status is the durable record).
func (s *Service) triggerDistill(profileID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), distillTimeout)
		defer cancel()
		if err := s.Distill(ctx, profileID); err != nil {
			s.logger.Printf("profiles: distill %s failed: %v", profileID, err)
		}
	}()
}

// Redistill sets a profile back to pending and re-triggers distillation. It
// powers a manual "re-run" endpoint. Returns ErrNotFound for an unknown id.
func (s *Service) Redistill(ctx context.Context, profileID string) error {
	if err := s.repo.SetStyleGuide(ctx, profileID, "", profile.StyleStatusPending, ""); err != nil {
		return err
	}
	s.runDistill(profileID)
	return nil
}

// RecoverPending re-triggers distillation for every profile left in the pending
// state (e.g. after a crash mid-distillation). Called once at startup.
func (s *Service) RecoverPending(ctx context.Context) {
	pending, err := s.repo.ListByStyleStatus(ctx, profile.StyleStatusPending)
	if err != nil {
		s.logger.Printf("profiles: recover pending: %v", err)
		return
	}
	for _, p := range pending {
		s.runDistill(p.ID)
	}
	if len(pending) > 0 {
		s.logger.Printf("profiles: re-triggered %d pending distillation(s)", len(pending))
	}
}
