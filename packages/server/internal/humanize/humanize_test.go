package humanize

import (
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

func sampleReview() review.Review {
	return review.Review{
		Summary: "Overall solid, a couple of issues.",
		Findings: []review.Finding{
			{
				Dimension: review.Risk, Severity: review.SeverityHigh,
				File: "auth.go", Line: 42, Issue: "hardcoded secret",
				Why: "leaks credentials", Fix: "read from env", Blocking: true,
			},
			{
				Dimension: review.Readability, Severity: review.SeverityLow,
				File: "util.go", Line: 0, Issue: "unclear name",
				Why: "hurts maintenance", Fix: "rename it",
			},
		},
	}
}

func TestBuildFindingMessages(t *testing.T) {
	const guide = "Voice: warm and terse. Uses voseo."
	f := sampleReview().Findings[0]
	msgs := BuildFindingMessages(guide, f)

	if len(msgs) != 2 {
		t.Fatalf("messages len = %d, want 2", len(msgs))
	}
	if msgs[0].Role != llm.RoleSystem || msgs[1].Role != llm.RoleUser {
		t.Fatalf("unexpected roles: %+v", msgs)
	}

	sys := msgs[0].Content
	if !strings.Contains(sys, guide) {
		t.Fatal("system prompt must include the style guide verbatim")
	}
	if !strings.Contains(sys, `"issue"`) || !strings.Contains(sys, `"why"`) || !strings.Contains(sys, `"fix"`) {
		t.Fatalf("system prompt must describe the issue/why/fix JSON contract, got: %q", sys)
	}

	user := msgs[1].Content
	for _, want := range []string{"hardcoded secret", "leaks credentials", "read from env", "auth.go:42"} {
		if !strings.Contains(user, want) {
			t.Fatalf("user prompt missing %q, got: %q", want, user)
		}
	}
}

func TestBuildSummaryMessages(t *testing.T) {
	const guide = "Voice: warm and terse. Uses voseo."
	msgs := BuildSummaryMessages(guide, sampleReview())

	if len(msgs) != 2 {
		t.Fatalf("messages len = %d, want 2", len(msgs))
	}
	if msgs[0].Role != llm.RoleSystem || msgs[1].Role != llm.RoleUser {
		t.Fatalf("unexpected roles: %+v", msgs)
	}

	sys := msgs[0].Content
	if !strings.Contains(sys, guide) {
		t.Fatal("system prompt must include the style guide verbatim")
	}
	if !strings.Contains(sys, `"summary"`) {
		t.Fatalf("system prompt must describe the summary JSON contract, got: %q", sys)
	}

	user := msgs[1].Content
	if !strings.Contains(user, "Overall solid, a couple of issues.") {
		t.Fatalf("user prompt must include the review summary, got: %q", user)
	}
}

func TestParseFindingHumanizedPlain(t *testing.T) {
	const content = `{"issue":"ojo con el secreto","why":"filtra credenciales","fix":"leelo del env"}`

	fh, err := ParseFindingHumanized(content)
	if err != nil {
		t.Fatalf("ParseFindingHumanized: %v", err)
	}
	if fh.Issue != "ojo con el secreto" || fh.Why != "filtra credenciales" || fh.Fix != "leelo del env" {
		t.Fatalf("parts not parsed: %+v", fh)
	}
}

func TestParseFindingHumanizedFenced(t *testing.T) {
	content := "```json\n" +
		`{"issue":"i","why":"","fix":"f"}` +
		"\n```"

	fh, err := ParseFindingHumanized(content)
	if err != nil {
		t.Fatalf("ParseFindingHumanized (fenced): %v", err)
	}
	if fh.Issue != "i" || fh.Why != "" || fh.Fix != "f" {
		t.Fatalf("fenced parse wrong: %+v", fh)
	}
}

func TestParseFindingHumanizedGarbage(t *testing.T) {
	if _, err := ParseFindingHumanized("not json at all"); err == nil {
		t.Fatal("expected an error on garbage input")
	}
}

func TestParseSummaryHumanizedPlain(t *testing.T) {
	const content = `{"summary":"che, en gral quedó lindo"}`

	sh, err := ParseSummaryHumanized(content)
	if err != nil {
		t.Fatalf("ParseSummaryHumanized: %v", err)
	}
	if sh.Summary != "che, en gral quedó lindo" {
		t.Fatalf("summary = %q", sh.Summary)
	}
}

func TestParseSummaryHumanizedFenced(t *testing.T) {
	content := "```json\n" + `{"summary":"buen laburo"}` + "\n```"

	sh, err := ParseSummaryHumanized(content)
	if err != nil {
		t.Fatalf("ParseSummaryHumanized (fenced): %v", err)
	}
	if sh.Summary != "buen laburo" {
		t.Fatalf("fenced parse wrong: %+v", sh)
	}
}

func TestParseSummaryHumanizedGarbage(t *testing.T) {
	if _, err := ParseSummaryHumanized("not json at all"); err == nil {
		t.Fatal("expected an error on garbage input")
	}
	if _, err := ParseSummaryHumanized(`{"summary":""}`); err == nil {
		t.Fatal("expected an error on empty summary")
	}
}
