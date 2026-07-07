package ai

import (
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
)

func TestFactorySelectsClient(t *testing.T) {
	openai, err := New(provider.Provider{Kind: provider.KindOpenAICompat}, "k")
	if err != nil {
		t.Fatalf("openai-compat: %v", err)
	}
	if _, ok := openai.(*OpenAIClient); !ok {
		t.Fatalf("openai-compat -> %T, want *OpenAIClient", openai)
	}

	claude, err := New(provider.Provider{Kind: provider.KindAnthropic}, "k")
	if err != nil {
		t.Fatalf("anthropic: %v", err)
	}
	if _, ok := claude.(*AnthropicClient); !ok {
		t.Fatalf("anthropic -> %T, want *AnthropicClient", claude)
	}
}

func TestFactoryRejectsUnknownKind(t *testing.T) {
	if _, err := New(provider.Provider{Kind: "bogus"}, "k"); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
