// Package mergerequest builds the LLM prompt that drafts a merge request's
// title and description from the diff between two branches — optionally in a
// profile author's voice — and parses the model's reply back into the two
// parts. It mirrors the internal/humanize prompt package: pure prompt/parse
// logic with no I/O, so the app service owns provider resolution and GitLab
// access.
package mergerequest

import (
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

// Generated is a drafted merge request: a one-line title and a Markdown
// description.
type Generated struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// FileDiff is one changed file's unified-diff text plus its change kind.
type FileDiff struct {
	Path   string
	Status string // "added" | "deleted" | "renamed" | "modified"
	Diff   string
}

// DiffInput is the branch-comparison material fed to the model.
type DiffInput struct {
	SourceBranch string
	TargetBranch string
	Commits      []string // commit subjects
	Files        []FileDiff
	Truncated    bool // true when Files was capped for size
}

// BuildMessages produces the system+user messages that ask the model to draft a
// merge request title and description from the diff. When styleGuide is
// non-empty the description is written in that author's voice and language;
// otherwise it is plain, professional English. Either way the response FORMAT
// (title line, blank line, Markdown body) is fixed so the reply parses cleanly.
func BuildMessages(styleGuide string, in DiffInput) []llm.Message {
	return []llm.Message{
		{Role: llm.RoleSystem, Content: systemPrompt(styleGuide)},
		{Role: llm.RoleUser, Content: userPrompt(in)},
	}
}

func systemPrompt(styleGuide string) string {
	var b strings.Builder
	b.WriteString("You write GitLab merge request titles and descriptions from a code diff.\n\n")

	if strings.TrimSpace(styleGuide) != "" {
		b.WriteString("Author style guide (apply it verbatim to the description's voice and language):\n")
		b.WriteString(styleGuide)
		b.WriteString("\n\n")
	} else {
		b.WriteString("Write in clear, professional English.\n\n")
	}

	b.WriteString(
		"Summarize what the change does and why, grounded ONLY in the diff and " +
			"commit messages provided. Never invent changes that are not present. " +
			"Aim to help a reviewer: a short summary of the intent, then the key " +
			"changes, and any notable impact or risk. Use Markdown in the " +
			"description.\n\n")

	b.WriteString("Respond in EXACTLY this shape and nothing else:\n")
	b.WriteString("- The FIRST line is the title on its own: a concise, imperative one-liner, no prefix, no Markdown heading.\n")
	b.WriteString("- Then ONE blank line.\n")
	b.WriteString("- Then the description, in Markdown.\n")
	return b.String()
}

func userPrompt(in DiffInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Merge request: %s → %s\n\n", in.SourceBranch, in.TargetBranch)

	if len(in.Commits) > 0 {
		b.WriteString("Commits:\n")
		for _, c := range in.Commits {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	b.WriteString("Diff:\n")
	if len(in.Files) == 0 {
		b.WriteString("(no file-level diff available; summarize from the commits)\n\n")
	}
	for _, f := range in.Files {
		fmt.Fprintf(&b, "### %s (%s)\n", f.Path, f.Status)
		b.WriteString("```diff\n")
		b.WriteString(f.Diff)
		if !strings.HasSuffix(f.Diff, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}
	if in.Truncated {
		b.WriteString("(diff truncated for length; summarize from what is shown)\n\n")
	}

	b.WriteString("Write the title and description now.")
	return b.String()
}

// Parse splits the model reply into a title (the first non-empty line) and a
// Markdown description (everything after it). It defends against a model that
// wraps the whole reply in a code fence, prefixes the title with a Markdown
// heading, or labels the parts "Title:"/"Description:".
func Parse(content string) (Generated, error) {
	s := strings.TrimSpace(content)
	s = stripOuterFence(s)
	if s == "" {
		return Generated{}, fmt.Errorf("mergerequest: model returned empty output")
	}

	lines := strings.Split(s, "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return Generated{}, fmt.Errorf("mergerequest: model returned no title")
	}

	title := stripLabel(strings.TrimSpace(strings.TrimLeft(lines[i], "# ")), "title")
	desc := strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
	desc = stripLabel(desc, "description")

	if title == "" {
		return Generated{}, fmt.Errorf("mergerequest: model returned no title")
	}
	return Generated{Title: title, Description: desc}, nil
}

// stripLabel removes a leading "<label>:" prefix (case-insensitive) from s.
func stripLabel(s, label string) string {
	if len(s) >= len(label)+1 &&
		strings.EqualFold(s[:len(label)], label) &&
		s[len(label)] == ':' {
		return strings.TrimSpace(s[len(label)+1:])
	}
	return s
}

// stripOuterFence unwraps a reply that the model wrapped entirely in a single
// ``` code fence.
func stripOuterFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:]
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
