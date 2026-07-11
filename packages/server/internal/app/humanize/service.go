// Package humanize rewrites a finished review's summary and findings in a
// profile's author voice, returning N complete variants. It is ephemeral: the
// variants are computed on demand from the default provider and never persisted.
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

// Count bounds: the model returns between 1 and 5 variants; 3 is the default.
const (
	defaultCount = 3
	minCount     = 1
	maxCount     = 5
)

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

// Humanize rewrites a finished review's summary and findings in the profile's
// author voice and returns count variants (clamped to [1,5], defaulting to 3).
// The review must be done and the profile's style guide must be ready. Variants
// are returned, not persisted.
func (s *Service) Humanize(ctx context.Context, reviewID, profileID string, count int) ([]humanize.Variant, error) {
	count = clampCount(count)

	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if rv.Status != review.StatusDone {
		return nil, fmt.Errorf("%w: cannot humanize a review in status %q", ErrReviewNotDone, rv.Status)
	}

	p, err := s.profiles.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if p.StyleGuideStatus != profile.StyleStatusReady {
		return nil, fmt.Errorf("%w (%s)", ErrStyleGuideNotReady, statusLabel(p.StyleGuideStatus))
	}

	variants, err := s.complete(ctx, p.StyleGuide, rv, count)
	if err != nil {
		return nil, err
	}
	return dropInvalidIndices(variants, len(rv.Findings)), nil
}

// complete resolves the default provider, builds the AI client, runs the single
// rewrite completion, and parses the returned variants.
func (s *Service) complete(ctx context.Context, styleGuide string, rv review.Review, count int) ([]humanize.Variant, error) {
	prov, err := s.providers.Default(ctx)
	if err != nil {
		return nil, err
	}
	apiKey, err := s.providers.APIKey(ctx, prov.ID)
	if err != nil {
		return nil, err
	}
	model := prov.Model
	if model == "" {
		return nil, fmt.Errorf("humanize: no model set on default provider %q", prov.Name)
	}
	client, err := ai.New(prov, apiKey)
	if err != nil {
		return nil, err
	}
	resp, err := client.Complete(ctx, llm.Request{
		Model:       model,
		Messages:    humanize.BuildHumanizeMessages(styleGuide, rv, count),
		Temperature: prov.Temperature,
	})
	if err != nil {
		return nil, err
	}
	return humanize.ParseVariants(resp.Content)
}

// clampCount defaults a non-positive count to 3 and clamps to [1,5].
func clampCount(count int) int {
	if count <= 0 {
		return defaultCount
	}
	if count < minCount {
		return minCount
	}
	if count > maxCount {
		return maxCount
	}
	return count
}

// dropInvalidIndices removes finding entries whose index is out of range for the
// review, defending against a hallucinated index the model may return.
func dropInvalidIndices(variants []humanize.Variant, findingCount int) []humanize.Variant {
	for i := range variants {
		kept := make([]humanize.FindingText, 0, len(variants[i].Findings))
		for _, f := range variants[i].Findings {
			if f.Index >= 0 && f.Index < findingCount {
				kept = append(kept, f)
			}
		}
		variants[i].Findings = kept
	}
	return variants
}

// statusLabel renders a style-guide status for an error message, mapping the
// empty "none" status to a readable word.
func statusLabel(status string) string {
	if status == profile.StyleStatusNone {
		return "none"
	}
	return status
}
