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

	// Gemini reuses the OpenAI-compat client, defaulting to Gemini's endpoint
	// when no base URL is set.
	gemini, err := New(provider.Provider{Kind: provider.KindGemini}, "k")
	if err != nil {
		t.Fatalf("gemini: %v", err)
	}
	gc, ok := gemini.(*OpenAIClient)
	if !ok {
		t.Fatalf("gemini -> %T, want *OpenAIClient", gemini)
	}
	if gc.baseURL != geminiOpenAIBaseURL {
		t.Fatalf("gemini baseURL = %q, want %q", gc.baseURL, geminiOpenAIBaseURL)
	}

	// An explicit base URL on a Gemini provider is honored (not overridden).
	custom, err := New(provider.Provider{Kind: provider.KindGemini, BaseURL: "https://proxy.test/v1"}, "k")
	if err != nil {
		t.Fatalf("gemini custom: %v", err)
	}
	if cc, ok := custom.(*OpenAIClient); !ok || cc.baseURL != "https://proxy.test/v1" {
		t.Fatalf("gemini custom baseURL not honored: %T %v", custom, custom)
	}

	// OpenRouter reuses the OpenAI-compat client, defaulting to OpenRouter's
	// endpoint when no base URL is set.
	openrouter, err := New(provider.Provider{Kind: provider.KindOpenRouter}, "k")
	if err != nil {
		t.Fatalf("openrouter: %v", err)
	}
	oc, ok := openrouter.(*OpenAIClient)
	if !ok {
		t.Fatalf("openrouter -> %T, want *OpenAIClient", openrouter)
	}
	if oc.baseURL != openRouterBaseURL {
		t.Fatalf("openrouter baseURL = %q, want %q", oc.baseURL, openRouterBaseURL)
	}

	// An explicit base URL on an OpenRouter provider is honored (not overridden).
	orCustom, err := New(provider.Provider{Kind: provider.KindOpenRouter, BaseURL: "https://proxy.test/or"}, "k")
	if err != nil {
		t.Fatalf("openrouter custom: %v", err)
	}
	if cc, ok := orCustom.(*OpenAIClient); !ok || cc.baseURL != "https://proxy.test/or" {
		t.Fatalf("openrouter custom baseURL not honored: %T %v", orCustom, orCustom)
	}
}

func TestFactoryRejectsUnknownKind(t *testing.T) {
	if _, err := New(provider.Provider{Kind: "bogus"}, "k"); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
