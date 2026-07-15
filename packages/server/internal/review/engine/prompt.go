package engine

import (
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

const frameworkPreamble = `You are a senior code reviewer applying the 4R quality framework to a merge request diff: Risk (R1), Readability (R2), Reliability (R3), Resilience (R4).

Review the diff systematically through each lens using the rules below. Report only genuine, concrete, located issues — never invent findings to seem thorough. If a dimension is clean, produce no findings for it.`

// precisionGate is a cross-cutting rule injected into the system prompt of
// BOTH review strategies (single-pass and multi-pass). It biases the model
// toward precision over recall so the 4R review does not drown authors in
// speculative or pre-existing nits. It is defined once here and shared; it is
// intentionally NOT duplicated into the per-dimension skill files.
const precisionGate = `PRECISION GATE — applies to every finding you report:
- Report a finding ONLY if you can point to the specific changed line(s) or symbol AND name a concrete, realistic failure or harm it causes. No hypotheticals, no "could be improved".
- Only flag issues INTRODUCED or exposed by THIS diff — never pre-existing code you happen to see.
- Style and preference findings are banned unless they obscure a genuine defect.
- Do NOT report speculative risks or anything you are not confident is a real problem. When in doubt, stay silent.
- Every finding's "why" must state the concrete impact, not a restatement of the rule.
- Weigh the cost: a missed nitpick costs nothing; a false positive burns the author's time and erodes trust in the review. Prefer silence.`

const jsonContract = `Respond with ONLY a JSON object, no prose, no markdown fences, matching exactly this schema:

{
  "summary": "3-5 sentence description of what the change does and its overall quality",
  "findings": [
    {
      "dimension": "risk | readability | reliability | resilience",
      "severity": "high | medium | low",
      "file": "path/to/file",
      "line": 0,
      "issue": "one or two sentences describing the problem",
      "why": "the technical impact if left unaddressed",
      "fix": "a specific, actionable suggestion",
      "blocking": true
    }
  ]
}

Use "line" 0 when a finding is not tied to a specific line. Set "blocking" true only per each dimension's blocking guidance. Return {"summary": "...", "findings": []} when the change is clean.`

// buildMessages assembles the system prompt (framework + rules + JSON contract)
// and the user prompt (the merge request content) for a single-pass review.
func buildMessages(set skills.Set, in Input) []llm.Message {
	system := strings.Join([]string{frameworkPreamble, set.Combined(), precisionGate, jsonContract}, "\n\n---\n\n")
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: renderUserPrompt(in)},
	}
}

// renderUserPrompt formats the merge request content shown to the model.
func renderUserPrompt(in Input) string {
	var user strings.Builder
	fmt.Fprintf(&user, "Merge request !%d", in.MRIID)
	if in.Title != "" {
		fmt.Fprintf(&user, ": %s", in.Title)
	}
	user.WriteString("\n\n")
	if in.Description != "" {
		fmt.Fprintf(&user, "Description:\n%s\n\n", in.Description)
	}
	user.WriteString("Diff:\n")
	user.WriteString(in.Diff)
	return user.String()
}
