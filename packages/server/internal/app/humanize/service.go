// Package humanize rewrites a finished review's parts in a profile's author
// voice, one target at a time. It is ephemeral: each rewrite is computed on
// demand from the default provider and never persisted. The frontend fires one
// call per target (a finding or the summary), so a rewrite always returns the
// structured parts of exactly one target.
package humanize

import (
	"context"
	"errors"
	"fmt"
	"log"

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
	reviews   review.Repository
	profiles  profile.Repository
	providers *providers.Service
	logger    *log.Logger
}

// NewService wires the humanize service. providers resolves the default LLM used
// for rewriting; logger defaults to log.Default() when nil.
func NewService(reviews review.Repository, profiles profile.Repository, providers *providers.Service, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{reviews: reviews, profiles: profiles, providers: providers, logger: logger}
}

// HumanizeFinding rewrites a single finding's issue/why/fix parts in the
// profile's author voice. The review must be done, the profile's style guide
// must be ready, and index must be within the review's findings. The rewritten
// parts are returned, not persisted.
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
	return humanize.ParseFindingHumanized(content)
}

// HumanizeSummary rewrites the review summary in the profile's author voice. The
// review must be done and the profile's style guide must be ready. The rewritten
// summary is returned, not persisted.
func (s *Service) HumanizeSummary(ctx context.Context, reviewID, profileID string) (humanize.SummaryHumanized, error) {
	rv, p, err := s.resolve(ctx, reviewID, profileID)
	if err != nil {
		return humanize.SummaryHumanized{}, err
	}

	content, err := s.complete(ctx, humanize.BuildSummaryMessages(p.StyleGuide, rv))
	if err != nil {
		return humanize.SummaryHumanized{}, err
	}
	return humanize.ParseSummaryHumanized(content)
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
