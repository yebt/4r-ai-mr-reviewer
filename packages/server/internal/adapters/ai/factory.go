package ai

import (
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
)

// geminiOpenAIBaseURL is Google Gemini's OpenAI-compatible endpoint, used when a
// Gemini provider leaves its base URL empty.
const geminiOpenAIBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

// New builds the llm.Client for a provider configuration and its API key.
func New(p provider.Provider, apiKey string) (llm.Client, error) {
	switch p.Kind {
	case provider.KindOpenAICompat:
		return NewOpenAIClient(p.BaseURL, apiKey), nil
	case provider.KindGemini:
		// Gemini speaks the OpenAI protocol at its own endpoint; default the base
		// URL to that endpoint when the provider didn't set one.
		baseURL := p.BaseURL
		if baseURL == "" {
			baseURL = geminiOpenAIBaseURL
		}
		return NewOpenAIClient(baseURL, apiKey), nil
	case provider.KindAnthropic:
		return NewAnthropicClient(p.BaseURL, apiKey), nil
	default:
		return nil, fmt.Errorf("ai: unsupported provider kind %q", p.Kind)
	}
}
