package humanize

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

// FindingText is one finding rewritten in the author's voice, tied back to the
// original finding by its zero-based index.
type FindingText struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

// Variant is a complete humanized rewrite of a review: the summary plus every
// finding, all rendered in the author's voice.
type Variant struct {
	Summary  string        `json:"summary"`
	Findings []FindingText `json:"findings"`
}

// BuildHumanizeMessages produces the system+user messages that ask the model to
// rewrite a finished review in the author's voice described by styleGuide,
// returning exactly count distinct complete variants as structured JSON.
func BuildHumanizeMessages(styleGuide string, rv review.Review, count int) []llm.Message {
	return []llm.Message{
		{Role: llm.RoleSystem, Content: humanizeSystemPrompt(styleGuide, count)},
		{Role: llm.RoleUser, Content: humanizeUserPrompt(rv)},
	}
}

func humanizeSystemPrompt(styleGuide string, count int) string {
	var b strings.Builder
	b.WriteString("You rewrite code-review feedback in a specific author's voice.\n\n")

	b.WriteString("Author style guide (apply it verbatim):\n")
	b.WriteString(styleGuide)
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf(
		"Produce exactly %d DISTINCT complete variants of the review. Each variant "+
			"rewrites the summary AND every finding into natural prose in the author's "+
			"voice and language as described by the style guide.\n", count))
	b.WriteString(
		"Preserve the technical substance of each finding (the issue, why it matters, " +
			"and the fix) — only change the VOICE and phrasing. Never invent, merge, drop, " +
			"or reorder findings. Rewrite every finding exactly once per variant and " +
			"reference it by its original `index`. Keep each finding's rewritten text " +
			"self-contained so it reads on its own.\n\n")

	b.WriteString("Respond with ONLY a JSON object, no prose and no markdown fences:\n")
	b.WriteString(`{ "variants": [ { "summary": "...", "findings": [ { "index": 0, "text": "..." } ] } ] }`)
	return b.String()
}

func humanizeUserPrompt(rv review.Review) string {
	var b strings.Builder
	b.WriteString("Review summary:\n")
	b.WriteString(rv.Summary)
	b.WriteString("\n\nFindings to rewrite (map each rewrite back to its index):\n\n")
	for i, f := range rv.Findings {
		fmt.Fprintf(&b, "Finding index %d:\n", i)
		fmt.Fprintf(&b, "- dimension: %s\n", f.Dimension)
		fmt.Fprintf(&b, "- severity: %s\n", f.Severity)
		fmt.Fprintf(&b, "- location: %s\n", location(f))
		fmt.Fprintf(&b, "- issue: %s\n", f.Issue)
		fmt.Fprintf(&b, "- why: %s\n", f.Why)
		fmt.Fprintf(&b, "- fix: %s\n", f.Fix)
		fmt.Fprintf(&b, "- blocking: %t\n\n", f.Blocking)
	}
	b.WriteString("Produce the variants now.")
	return b.String()
}

// location renders a finding's file:line, or just the file when it is not tied
// to a specific line (line 0).
func location(f review.Finding) string {
	if f.File == "" {
		return "(general)"
	}
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return f.File
}

// variantsDTO mirrors the JSON contract the model is asked to return.
type variantsDTO struct {
	Variants []Variant `json:"variants"`
}

// ParseVariants extracts and decodes the model's JSON into variants. It tolerates
// responses wrapped in markdown code fences or padded with surrounding prose.
// Parsing is pure: it does not range-validate finding indices against any review
// (the service does that, since it holds the review). It returns an error when no
// variants can be parsed.
func ParseVariants(content string) ([]Variant, error) {
	raw := extractJSON(content)

	var dto variantsDTO
	if err := json.Unmarshal([]byte(raw), &dto); err != nil {
		return nil, fmt.Errorf("humanize: decode model output: %w", err)
	}
	if len(dto.Variants) == 0 {
		return nil, fmt.Errorf("humanize: model returned no variants")
	}
	return dto.Variants, nil
}

// extractJSON pulls the JSON object out of a model response that may include
// code fences or leading/trailing prose.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
