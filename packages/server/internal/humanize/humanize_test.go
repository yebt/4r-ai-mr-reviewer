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

func TestBuildHumanizeMessages(t *testing.T) {
	const guide = "Voice: warm and terse. Uses voseo."
	msgs := BuildHumanizeMessages(guide, sampleReview(), 3)

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
	if !strings.Contains(sys, "exactly 3") {
		t.Fatalf("system prompt must state the count instruction, got: %q", sys)
	}
	if !strings.Contains(sys, `"variants"`) {
		t.Fatal("system prompt must describe the variants JSON contract")
	}

	user := msgs[1].Content
	for _, want := range []string{"hardcoded secret", "unclear name", "index 0", "index 1", "auth.go:42"} {
		if !strings.Contains(user, want) {
			t.Fatalf("user prompt missing %q, got: %q", want, user)
		}
	}
}

func TestParseVariantsPlain(t *testing.T) {
	const content = `{"variants":[
		{"summary":"che, quedó lindo","findings":[{"index":0,"text":"ojo con el secreto"}]},
		{"summary":"buen laburo","findings":[{"index":0,"text":"sacá el secreto"},{"index":1,"text":"renombralo"}]}
	]}`

	vs, err := ParseVariants(content)
	if err != nil {
		t.Fatalf("ParseVariants: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("variants len = %d, want 2", len(vs))
	}
	if vs[0].Summary != "che, quedó lindo" {
		t.Fatalf("summary = %q", vs[0].Summary)
	}
	if len(vs[1].Findings) != 2 || vs[1].Findings[1].Index != 1 || vs[1].Findings[1].Text != "renombralo" {
		t.Fatalf("findings not parsed: %+v", vs[1].Findings)
	}
}

func TestParseVariantsFenced(t *testing.T) {
	content := "```json\n" +
		`{"variants":[{"summary":"s","findings":[{"index":0,"text":"t"}]}]}` +
		"\n```"

	vs, err := ParseVariants(content)
	if err != nil {
		t.Fatalf("ParseVariants (fenced): %v", err)
	}
	if len(vs) != 1 || vs[0].Summary != "s" || vs[0].Findings[0].Text != "t" {
		t.Fatalf("fenced parse wrong: %+v", vs)
	}
}

func TestParseVariantsGarbage(t *testing.T) {
	if _, err := ParseVariants("not json at all"); err == nil {
		t.Fatal("expected an error on garbage input")
	}
	if _, err := ParseVariants(`{"variants":[]}`); err == nil {
		t.Fatal("expected an error when no variants are present")
	}
}
