// Package humanize distills a user's raw writing samples into a compact,
// reusable style guide that another model can later apply to rewrite text in
// that author's voice. It owns only the prompt construction; the LLM call and
// persistence live in the profiles service.
package humanize

import (
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
)

// BuildDistillMessages produces the system+user messages for distilling a
// profile's writing samples into a style guide. The system prompt fixes the
// task and honours the profile's knobs (language, formality, emoji usage); the
// user message carries the numbered samples.
func BuildDistillMessages(p profile.Profile) []llm.Message {
	return []llm.Message{
		{Role: llm.RoleSystem, Content: distillSystemPrompt(p)},
		{Role: llm.RoleUser, Content: distillUserPrompt(p)},
	}
}

func distillSystemPrompt(p profile.Profile) string {
	var b strings.Builder
	b.WriteString(
		"You are a writing-style analyst. Study the author's writing samples below and produce a " +
			"COMPACT, reusable style guide that another model can later apply to rewrite arbitrary text " +
			"in this author's voice. Capture the concrete, transferable traits of the voice: tone, " +
			"register, typical sentence length and rhythm, vocabulary and favourite words, recurring " +
			"phrasings or quirks, and punctuation habits.\n\n")

	b.WriteString("Honour these constraints:\n")
	b.WriteString(fmt.Sprintf("- Target language: %s.\n", knobValue(p.Language, "match the language of the samples")))
	b.WriteString(fmt.Sprintf("- Formality: %s.\n", knobValue(p.Formality, "match the formality of the samples")))
	if p.Emojis {
		b.WriteString("- Emojis: the voice USES emojis; note where and how they appear.\n")
	} else {
		b.WriteString("- Emojis: the voice does NOT use emojis; the guide must instruct to avoid them.\n")
	}

	b.WriteString(
		"\nOutput PLAIN TEXT only: no JSON, no markdown fences, no preamble. A few short paragraphs or " +
			"terse bullet lines. Be specific and deterministic; describe the voice, do not rewrite the samples.")
	return b.String()
}

func distillUserPrompt(p profile.Profile) string {
	var b strings.Builder
	b.WriteString("Writing samples from the author:\n\n")
	for i, s := range p.Samples {
		b.WriteString(fmt.Sprintf("Sample %d:\n%s\n\n", i+1, s))
	}
	b.WriteString("Produce the style guide now.")
	return b.String()
}

// knobValue returns v when set, otherwise a fallback instruction.
func knobValue(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
