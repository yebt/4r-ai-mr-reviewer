package ai

import (
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
)

// New builds the llm.Client for a provider configuration and its API key.
func New(p provider.Provider, apiKey string) (llm.Client, error) {
	switch p.Kind {
	case provider.KindOpenAICompat:
		return NewOpenAIClient(p.BaseURL, apiKey), nil
	case provider.KindAnthropic:
		return NewAnthropicClient(p.BaseURL, apiKey), nil
	default:
		return nil, fmt.Errorf("ai: unsupported provider kind %q", p.Kind)
	}
}
