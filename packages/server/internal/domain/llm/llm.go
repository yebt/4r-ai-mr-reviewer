// Package llm defines the provider-agnostic contract for talking to a large
// language model. Adapters (OpenAI-compatible, Anthropic) implement Client;
// the review engine depends only on this interface.
package llm

import "context"

// Role identifies who authored a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single turn in a conversation.
type Message struct {
	Role    Role
	Content string
}

// Request is a completion request. Temperature is optional: nil means "don't
// send it", so the model uses its own default (some models reject any other
// value). MaxTokens of 0 lets the adapter apply its default.
type Request struct {
	Model       string
	Messages    []Message
	Temperature *float64
	MaxTokens   int
	// ThinkingBudget requests the model expose its reasoning / chain-of-thought,
	// budgeting up to this many tokens for it. 0 disables reasoning capture.
	// Adapters that cannot surface reasoning ignore it.
	ThinkingBudget int
}

// Response is the model's reply plus token accounting.
type Response struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Model        string
	// Reasoning is the model's exposed chain-of-thought for this call, when the
	// provider returned any and ThinkingBudget was set. Empty otherwise.
	Reasoning string
}

// Client sends a completion request to an AI provider.
type Client interface {
	Complete(ctx context.Context, req Request) (Response, error)
}
