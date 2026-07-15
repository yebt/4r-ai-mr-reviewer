// Package humanize rewrites a finished review's parts in a profile's author
// voice, one target at a time. Each rewrite is computed from the default
// provider and persisted (see HumanizationRepository) so tabs survive a reload.
// The frontend fires one call per target (a finding or the summary), so a
// rewrite always returns the structured parts of exactly one target.
package humanize

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/ai"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/humanize"
)

// ErrReviewNotDone is returned when humanization is requested for a review that
// has not finished. The HTTP layer maps it to 409 Conflict.
var ErrReviewNotDone = errors.New("humanize: review is not done")

// ErrStyleGuideNotReady is returned when the profile's style guide has not been
// distilled yet. The HTTP layer maps it to 409 Conflict.
var ErrStyleGuideNotReady = errors.New("humanize: style guide not ready")

// ErrFindingIndexOutOfRange is returned when the requested finding index does
// not exist in the review. The HTTP layer maps it to 400 Bad Request.
var ErrFindingIndexOutOfRange = errors.New("humanize: finding index out of range")

// Service rewrites finished reviews in an author's voice.
type Service struct {
	reviews       review.Repository
	profiles      profile.Repository
	humanizations review.HumanizationRepository
	providers     *providers.Service
	logger        *log.Logger
}

// NewService wires the humanize service. providers resolves the default LLM used
// for rewriting; humanizations persists each successful rewrite so it survives a
// page reload; logger defaults to log.Default() when nil.
func NewService(reviews review.Repository, profiles profile.Repository, humanizations review.HumanizationRepository, providers *providers.Service, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{reviews: reviews, profiles: profiles, humanizations: humanizations, providers: providers, logger: logger}
}

// HumanizeFinding rewrites a single finding's issue/why/fix parts in the
// profile's author voice. The review must be done, the profile's style guide
// must be ready, and index must be within the review's findings. The rewritten
// parts are returned and persisted as a new tab so they survive a page reload.
func (s *Service) HumanizeFinding(ctx context.Context, reviewID, profileID string, index int) (humanize.FindingHumanized, error) {
	rv, p, err := s.resolve(ctx, reviewID, profileID)
	if err != nil {
		return humanize.FindingHumanized{}, err
	}
	if index < 0 || index >= len(rv.Findings) {
		return humanize.FindingHumanized{}, fmt.Errorf("%w: index %d, %d finding(s)", ErrFindingIndexOutOfRange, index, len(rv.Findings))
	}

	content, err := s.complete(ctx, humanize.BuildFindingMessages(p.StyleGuide, rv.Findings[index]))
	if err != nil {
		return humanize.FindingHumanized{}, err
	}
	fh, err := humanize.ParseFindingHumanized(content)
	if err != nil {
		return humanize.FindingHumanized{}, err
	}
	// Persist the run so a reload rehydrates it. The rewrite is already computed
	// and paid for, so a persistence failure must not discard it: persist retries,
	// then degrades to returning the unsaved text.
	s.persist(ctx, review.Humanization{
		ReviewID: reviewID, ProfileID: profileID, Target: review.HumanizationFinding,
		FindingIndex: index, Issue: fh.Issue, Why: fh.Why, Fix: fh.Fix,
	})
	return fh, nil
}

// HumanizeSummary rewrites the review summary in the profile's author voice. The
// review must be done and the profile's style guide must be ready. The rewritten
// summary is returned and persisted as a new tab so it survives a page reload.
func (s *Service) HumanizeSummary(ctx context.Context, reviewID, profileID string) (humanize.SummaryHumanized, error) {
	rv, p, err := s.resolve(ctx, reviewID, profileID)
	if err != nil {
		return humanize.SummaryHumanized{}, err
	}

	content, err := s.complete(ctx, humanize.BuildSummaryMessages(p.StyleGuide, rv))
	if err != nil {
		return humanize.SummaryHumanized{}, err
	}
	sh, err := humanize.ParseSummaryHumanized(content)
	if err != nil {
		return humanize.SummaryHumanized{}, err
	}
	// Persist the run so a reload rehydrates it. The rewrite is already computed
	// and paid for, so a persistence failure must not discard it: persist retries,
	// then degrades to returning the unsaved text.
	s.persist(ctx, review.Humanization{
		ReviewID: reviewID, ProfileID: profileID, Target: review.HumanizationSummary,
		FindingIndex: review.SummaryFindingIndex, Summary: sh.Summary,
	})
	return sh, nil
}

// persist records a humanize run, retrying briefly to ride out a transient DB
// failure. The run is already computed and paid for, so on a persistent failure
// it logs and returns without error: the caller still gets the text this session,
// it just won't survive a reload. Losing durability beats losing the paid output.
func (s *Service) persist(ctx context.Context, h review.Humanization) {
	const attempts = 3
	var err error
	for i := 0; i < attempts; i++ {
		if _, err = s.humanizations.Add(ctx, h); err == nil {
			return
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			s.logger.Printf("humanize: persist %s for review %s: %v", h.Target, h.ReviewID, ctx.Err())
			return
		case <-time.After(time.Duration(i+1) * 50 * time.Millisecond):
		}
	}
	s.logger.Printf("humanize: persist %s for review %s failed, returning unsaved text: %v", h.Target, h.ReviewID, err)
}

// List returns every persisted humanization for a review so the HTTP layer can
// group them for the SPA to rehydrate its tabs.
func (s *Service) List(ctx context.Context, reviewID string) ([]review.Humanization, error) {
	return s.humanizations.ListByReview(ctx, reviewID)
}

// resolve loads and guards the review and profile shared by both humanize paths:
// the review must be done and the profile's style guide must be ready.
func (s *Service) resolve(ctx context.Context, reviewID, profileID string) (review.Review, profile.Profile, error) {
	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return review.Review{}, profile.Profile{}, err
	}
	if rv.Status != review.StatusDone {
		return review.Review{}, profile.Profile{}, fmt.Errorf("%w: cannot humanize a review in status %q", ErrReviewNotDone, rv.Status)
	}

	p, err := s.profiles.Get(ctx, profileID)
	if err != nil {
		return review.Review{}, profile.Profile{}, err
	}
	if p.StyleGuideStatus != profile.StyleStatusReady {
		return review.Review{}, profile.Profile{}, fmt.Errorf("%w (%s)", ErrStyleGuideNotReady, statusLabel(p.StyleGuideStatus))
	}
	return rv, p, nil
}

// complete resolves the default provider, builds the AI client, runs the single
// rewrite completion, and returns the raw model content for the caller to parse.
func (s *Service) complete(ctx context.Context, msgs []llm.Message) (string, error) {
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
		return "", fmt.Errorf("humanize: no model set on default provider %q", prov.Name)
	}
	client, err := ai.New(prov, apiKey)
	if err != nil {
		return "", err
	}
	resp, err := client.Complete(ctx, llm.Request{
		Model:       model,
		Messages:    msgs,
		Temperature: prov.Temperature,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// statusLabel renders a style-guide status for an error message, mapping the
// empty "none" status to a readable word.
func statusLabel(status string) string {
	if status == profile.StyleStatusNone {
		return "none"
	}
	return status
}
