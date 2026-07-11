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

// Strategy runs a review, reporting fine-grained progress through onPhase
// (which may be nil). Single-pass and multi-pass both implement it.
type Strategy interface {
	Run(ctx context.Context, client llm.Client, model string, temperature *float64, in Input, onPhase func(phase string)) (review.Review, error)
}

// reportPhase calls onPhase if it is set.
func reportPhase(onPhase func(string), phase string) {
	if onPhase != nil {
		onPhase(phase)
	}
}

// Engine runs single-pass reviews with a fixed rule set.
type Engine struct {
	skills    skills.Set
	maxTokens int
}

// New builds an engine over a loaded rule set.
func New(set skills.Set) *Engine {
	return &Engine{skills: set}
}

var _ Strategy = (*Engine)(nil)

// Run executes one review against client using model, returning a completed
// Review. temperature is optional (nil = let the model use its default).
// Persistence and job orchestration are the caller's concern.
func (e *Engine) Run(ctx context.Context, client llm.Client, model string, temperature *float64, in Input, onPhase func(phase string)) (review.Review, error) {
	if in.Diff == "" {
		return review.Review{}, fmt.Errorf("engine: empty diff")
	}
	// Cooperative cancellation before the (single) LLM call.
	if err := ctx.Err(); err != nil {
		return review.Review{}, err
	}
	reportPhase(onPhase, "reviewing")

	resp, err := client.Complete(ctx, llm.Request{
		Model:       model,
		Messages:    buildMessages(e.skills, in),
		Temperature: temperature,
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
