package humanize

import (
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
)

func TestBuildDistillMessagesIncludesSamplesAndKnobs(t *testing.T) {
	p := profile.Profile{
		Language:  "es-AR",
		Formality: "casual",
		Emojis:    true,
		Samples:   []string{"che boludo todo bien", "dale nos vemos manana"},
	}

	msgs := BuildDistillMessages(p)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != llm.RoleSystem || msgs[1].Role != llm.RoleUser {
		t.Fatalf("unexpected roles: %v, %v", msgs[0].Role, msgs[1].Role)
	}

	system := msgs[0].Content
	for _, want := range []string{"es-AR", "casual"} {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt missing knob %q:\n%s", want, system)
		}
	}
	if !strings.Contains(strings.ToLower(system), "emoji") {
		t.Errorf("system prompt should mention emojis:\n%s", system)
	}

	user := msgs[1].Content
	for _, s := range p.Samples {
		if !strings.Contains(user, s) {
			t.Errorf("user prompt missing sample %q:\n%s", s, user)
		}
	}
	if !strings.Contains(user, "Sample 1") || !strings.Contains(user, "Sample 2") {
		t.Errorf("user prompt should number samples:\n%s", user)
	}
}

func TestBuildDistillMessagesNoEmojiInstruction(t *testing.T) {
	p := profile.Profile{Language: "en", Formality: "formal", Emojis: false, Samples: []string{"Dear team,"}}
	system := BuildDistillMessages(p)[0].Content
	if !strings.Contains(strings.ToLower(system), "does not use emojis") {
		t.Errorf("expected a no-emoji instruction:\n%s", system)
	}
}
