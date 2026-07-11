package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// MultiPass reviews each 4R dimension in its own focused call, reporting the
// current lens as a phase (for live progress). Findings are aggregated and
// scored deterministically; the summary is composed locally (no extra call).
type MultiPass struct {
	skills skills.Set
}

// NewMultiPass builds a multi-pass strategy over a loaded rule set.
func NewMultiPass(set skills.Set) *MultiPass {
	return &MultiPass{skills: set}
}

var _ Strategy = (*MultiPass)(nil)

type passSpec struct {
	phase string
	dim   review.Dimension
	skill string
}

// Run executes one focused call per dimension, in order.
func (m *MultiPass) Run(ctx context.Context, client llm.Client, model string, temperature *float64, in Input, onPhase func(phase string)) (review.Review, error) {
	if in.Diff == "" {
		return review.Review{}, fmt.Errorf("engine: empty diff")
	}

	passes := []passSpec{
		{"risk", review.Risk, m.skills.Risk},
		{"readability", review.Readability, m.skills.Readability},
		{"reliability", review.Reliability, m.skills.Reliability},
		{"resilience", review.Resilience, m.skills.Resilience},
	}

	var (
		findings     []review.Finding
		inputTokens  int
		outputTokens int
	)
	for _, p := range passes {
		// Cooperative cancellation between passes, in case the AI adapter did
		// not honour ctx on the in-flight call.
		if err := ctx.Err(); err != nil {
			return review.Review{}, err
		}
		reportPhase(onPhase, p.phase)

		resp, err := client.Complete(ctx, llm.Request{
			Model:       model,
			Messages:    buildDimensionMessages(p.dim, p.skill, in),
			Temperature: temperature,
		})
		if err != nil {
			return review.Review{}, err
		}
		inputTokens += resp.InputTokens
		outputTokens += resp.OutputTokens

		_, fs, err := parseResponse(resp.Content)
		if err != nil {
			return review.Review{}, fmt.Errorf("engine: %s pass: %w", p.phase, err)
		}
		// The prompt is dimension-specific; pin the dimension defensively.
		for i := range fs {
			fs[i].Dimension = p.dim
		}
		findings = append(findings, fs...)
	}

	rec, score := review.Evaluate(findings)
	return review.Review{
		RepoID:         in.RepoID,
		MRIID:          in.MRIID,
		Status:         review.StatusDone,
		Summary:        localSummary(findings),
		Findings:       findings,
		Recommendation: rec,
		Score:          score,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
	}, nil
}

func buildDimensionMessages(dim review.Dimension, skill string, in Input) []llm.Message {
	system := strings.Join([]string{dimensionPreamble(dim), skill, dimensionContract}, "\n\n---\n\n")
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: renderUserPrompt(in)},
	}
}

func dimensionPreamble(dim review.Dimension) string {
	return fmt.Sprintf(
		"You are a senior code reviewer examining a merge request diff through ONE lens only: %s. "+
			"Apply the rules below and report only genuine, concrete, located issues for this lens. "+
			"Do not invent findings; if the change is clean for this lens, produce none.",
		dimensionTitle(dim),
	)
}

func dimensionTitle(d review.Dimension) string {
	switch d {
	case review.Risk:
		return "R1 Risk"
	case review.Readability:
		return "R2 Readability"
	case review.Reliability:
		return "R3 Reliability"
	case review.Resilience:
		return "R4 Resilience"
	default:
		return string(d)
	}
}

const dimensionContract = `Respond with ONLY a JSON object, no prose, no markdown fences:

{
  "findings": [
    { "severity": "high | medium | low", "file": "path/to/file", "line": 0, "issue": "...", "why": "...", "fix": "...", "blocking": true }
  ]
}

Omit the "dimension" field. Use "line" 0 when a finding is not tied to a specific line. Return {"findings": []} when clean.`

func localSummary(findings []review.Finding) string {
	if len(findings) == 0 {
		return "No issues found across the 4R lenses — the change looks clean."
	}
	counts := map[review.Dimension]int{}
	blocking := 0
	for _, f := range findings {
		counts[f.Dimension]++
		if f.Blocking {
			blocking++
		}
	}
	return fmt.Sprintf(
		"Found %d finding(s) across the 4R lenses: %d risk, %d readability, %d reliability, %d resilience (%d blocking).",
		len(findings), counts[review.Risk], counts[review.Readability], counts[review.Reliability], counts[review.Resilience], blocking,
	)
}
