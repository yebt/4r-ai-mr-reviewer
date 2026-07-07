// Package engine runs a single-pass 4R review: it builds the prompt from the
// loaded skills and the merge request context, calls the LLM, parses the
// structured findings, and derives a deterministic score.
//
// It is the ReviewStrategy for the MVP. A future multi-pass strategy (four
// focused calls plus synthesis, with prompt caching) can implement the same
// Run contract without touching callers.
package engine

import (
	"context"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// Input is the merge request material a review runs against. The ContextStrategy
// (fast diff / deep-lite clone) is responsible for producing it.
type Input struct {
	RepoID      string
	MRIID       int
	Title       string
	Description string
	Diff        string
}

// Engine runs single-pass reviews with a fixed rule set.
type Engine struct {
	skills      skills.Set
	temperature float64
	maxTokens   int
}

// New builds an engine over a loaded rule set. Temperature defaults to 0 for
// deterministic reviews.
func New(set skills.Set) *Engine {
	return &Engine{skills: set}
}

// Run executes one review against client using model, returning a completed
// Review. Persistence and job orchestration are the caller's concern.
func (e *Engine) Run(ctx context.Context, client llm.Client, model string, in Input) (review.Review, error) {
	if in.Diff == "" {
		return review.Review{}, fmt.Errorf("engine: empty diff")
	}

	resp, err := client.Complete(ctx, llm.Request{
		Model:       model,
		Messages:    buildMessages(e.skills, in),
		Temperature: e.temperature,
		MaxTokens:   e.maxTokens,
	})
	if err != nil {
		return review.Review{}, err
	}

	summary, findings, err := parseResponse(resp.Content)
	if err != nil {
		return review.Review{}, fmt.Errorf("engine: %w", err)
	}

	rec, score := review.Evaluate(findings)
	return review.Review{
		RepoID:         in.RepoID,
		MRIID:          in.MRIID,
		Status:         review.StatusDone,
		Summary:        summary,
		Findings:       findings,
		Recommendation: rec,
		Score:          score,
		InputTokens:    resp.InputTokens,
		OutputTokens:   resp.OutputTokens,
	}, nil
}
