package engine

import (
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

const frameworkPreamble = `You are a senior code reviewer applying the 4R quality framework to a merge request diff: Risk (R1), Readability (R2), Reliability (R3), Resilience (R4).

Review the diff systematically through each lens using the rules below. Report only genuine, concrete, located issues — never invent findings to seem thorough. If a dimension is clean, produce no findings for it.`

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
	system := strings.Join([]string{frameworkPreamble, set.Combined(), jsonContract}, "\n\n---\n\n")

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

	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: user.String()},
	}
}
