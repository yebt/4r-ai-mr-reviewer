package humanize

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

// FindingHumanized is a single finding rewritten in the author's voice, keeping
// the issue/why/fix parts separate so the caller can reassemble them without
// losing any part of the original finding.
type FindingHumanized struct {
	Issue string `json:"issue"`
	Why   string `json:"why"`
	Fix   string `json:"fix"`
}

// SummaryHumanized is a review summary rewritten in the author's voice.
type SummaryHumanized struct {
	Summary string `json:"summary"`
}

// BuildFindingMessages produces the system+user messages that ask the model to
// rewrite a single finding's parts (issue/why/fix) in the author's voice
// described by styleGuide, returning the parts as structured JSON.
func BuildFindingMessages(styleGuide string, f review.Finding) []llm.Message {
	return []llm.Message{
		{Role: llm.RoleSystem, Content: findingSystemPrompt(styleGuide)},
		{Role: llm.RoleUser, Content: findingUserPrompt(f)},
	}
}

func findingSystemPrompt(styleGuide string) string {
	var b strings.Builder
	b.WriteString("You rewrite code-review feedback in a specific author's voice.\n\n")

	b.WriteString("Author style guide (apply it verbatim):\n")
	b.WriteString(styleGuide)
	b.WriteString("\n\n")

	b.WriteString(
		"Rewrite THIS SINGLE finding's three parts — the issue, why it matters, " +
			"and the fix — into natural prose in the author's voice and language as " +
			"described by the style guide. Preserve the technical substance of each " +
			"part exactly; only change the VOICE and phrasing. Never invent, merge, or " +
			"move content between parts. If a part is empty in the input, keep it " +
			"empty in the output — do not fabricate content for it.\n\n")

	b.WriteString("Respond with ONLY a JSON object, no prose and no markdown fences:\n")
	b.WriteString(`{ "issue": "...", "why": "...", "fix": "..." }`)
	return b.String()
}

func findingUserPrompt(f review.Finding) string {
	var b strings.Builder
	b.WriteString("Finding to rewrite:\n")
	fmt.Fprintf(&b, "- dimension: %s\n", f.Dimension)
	fmt.Fprintf(&b, "- severity: %s\n", f.Severity)
	fmt.Fprintf(&b, "- location: %s\n", location(f))
	fmt.Fprintf(&b, "- issue: %s\n", f.Issue)
	fmt.Fprintf(&b, "- why: %s\n", f.Why)
	fmt.Fprintf(&b, "- fix: %s\n\n", f.Fix)
	b.WriteString("Rewrite the issue, why and fix now. Keep an empty part empty.")
	return b.String()
}

// BuildSummaryMessages produces the system+user messages that ask the model to
// rewrite a review's summary in the author's voice described by styleGuide,
// returning it as structured JSON.
func BuildSummaryMessages(styleGuide string, rv review.Review) []llm.Message {
	return []llm.Message{
		{Role: llm.RoleSystem, Content: summarySystemPrompt(styleGuide)},
		{Role: llm.RoleUser, Content: summaryUserPrompt(rv)},
	}
}

func summarySystemPrompt(styleGuide string) string {
	var b strings.Builder
	b.WriteString("You rewrite code-review feedback in a specific author's voice.\n\n")

	b.WriteString("Author style guide (apply it verbatim):\n")
	b.WriteString(styleGuide)
	b.WriteString("\n\n")

	b.WriteString(
		"Rewrite the review summary into natural prose in the author's voice and " +
			"language as described by the style guide. Preserve the technical " +
			"substance; only change the VOICE and phrasing. Do not invent findings or " +
			"facts that are not in the summary.\n\n")

	b.WriteString("Respond with ONLY a JSON object, no prose and no markdown fences:\n")
	b.WriteString(`{ "summary": "..." }`)
	return b.String()
}

func summaryUserPrompt(rv review.Review) string {
	var b strings.Builder
	b.WriteString("Review summary to rewrite:\n")
	b.WriteString(rv.Summary)
	fmt.Fprintf(&b, "\n\n(Context: the review has %d finding(s).)\n\n", len(rv.Findings))
	b.WriteString("Rewrite the summary now.")
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

// ParseFindingHumanized extracts and decodes the model's JSON into the three
// rewritten finding parts. It tolerates responses wrapped in markdown code
// fences or padded with surrounding prose, and errors when nothing parses.
func ParseFindingHumanized(content string) (FindingHumanized, error) {
	raw := extractJSON(content)

	var out FindingHumanized
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return FindingHumanized{}, fmt.Errorf("humanize: decode model output: %w", err)
	}
	return out, nil
}

// ParseSummaryHumanized extracts and decodes the model's JSON into the rewritten
// summary. It tolerates code fences or surrounding prose and errors when nothing
// parses or the summary is empty.
func ParseSummaryHumanized(content string) (SummaryHumanized, error) {
	raw := extractJSON(content)

	var out SummaryHumanized
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return SummaryHumanized{}, fmt.Errorf("humanize: decode model output: %w", err)
	}
	if strings.TrimSpace(out.Summary) == "" {
		return SummaryHumanized{}, fmt.Errorf("humanize: model returned an empty summary")
	}
	return out, nil
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
