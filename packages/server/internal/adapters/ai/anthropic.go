package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicVersion        = "2023-06-01"
	// anthropicDefaultMaxTokens is used when the request leaves MaxTokens 0,
	// since the messages API requires the field.
	anthropicDefaultMaxTokens = 4096
	// anthropicMinThinkingBudget is the smallest budget the extended-thinking API
	// accepts; smaller requests are raised to it.
	anthropicMinThinkingBudget = 1024
)

// AnthropicClient talks to the Anthropic /v1/messages API (Claude).
type AnthropicClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewAnthropicClient builds a client. An empty baseURL falls back to Anthropic's.
func NewAnthropicClient(baseURL, apiKey string) *AnthropicClient {
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	return &AnthropicClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}
}

var _ llm.Client = (*AnthropicClient)(nil)

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicThinking enables extended thinking (reasoning) on a request.
type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	Thinking    *anthropicThinking `json:"thinking,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Thinking string `json:"thinking"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete sends a messages request. System-role messages are hoisted into the
// top-level system field, as the Anthropic API requires.
//
// When req.ThinkingBudget is set, extended thinking is enabled and the model's
// reasoning is captured into Response.Reasoning. Two thinking-specific failure
// modes each trigger a single retry with thinking disabled (never both — after
// any thinking-off retry the request no longer carries thinking, so neither
// branch can re-fire), so a reasoning-capture attempt never breaks a review:
//
//   - a 400 whose body points at a thinking/budget/temperature problem (a model
//     that does not support thinking, or a conflict);
//   - a valid 200 that carries only thinking blocks and no text, which happens
//     when the model exhausts max_tokens while reasoning.
func (c *AnthropicClient) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	body := c.buildRequest(req)

	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": anthropicVersion,
	}

	var out anthropicResponse
	err := postJSON(ctx, c.http, "anthropic", c.baseURL+"/v1/messages", headers, body, &out)
	if err != nil && body.Thinking != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest && isThinkingError(apiErr.Body) {
			// Thinking rejected (unsupported, or a thinking/budget/temperature
			// conflict): retry once without it, restoring the original temperature.
			// Any other 400 (bad model, malformed request) surfaces unchanged.
			log.Printf("ai: anthropic thinking rejected (400), retrying without thinking")
			out = anthropicResponse{}
			body, err = c.retryWithoutThinking(ctx, req, headers, &out)
		}
	}
	if err != nil {
		return llm.Response{}, err
	}

	text, reasoning := parseAnthropicContent(out)

	// A thinking-enabled 200 can come back with only thinking blocks and no text
	// when the model exhausts max_tokens while reasoning. Retry once without
	// thinking so the pass still gets a real answer. This cannot compound with the
	// 400 retry above: after any thinking-off retry body.Thinking is nil.
	if text == "" && body.Thinking != nil {
		log.Printf("ai: anthropic returned no text under thinking budget, retrying without thinking")
		out = anthropicResponse{}
		if body, err = c.retryWithoutThinking(ctx, req, headers, &out); err != nil {
			return llm.Response{}, err
		}
		text, reasoning = parseAnthropicContent(out)
	}

	if text == "" {
		return llm.Response{}, fmt.Errorf("ai: anthropic returned no text content")
	}
	return llm.Response{
		Content:      text,
		InputTokens:  out.Usage.InputTokens,
		OutputTokens: out.Usage.OutputTokens,
		Model:        out.Model,
		Reasoning:    reasoning,
	}, nil
}

// retryWithoutThinking re-sends req with the thinking budget cleared, decoding
// into out. It returns the wire request it sent so the caller can observe that
// thinking is now off (body.Thinking == nil) and avoid a second retry.
func (c *AnthropicClient) retryWithoutThinking(ctx context.Context, req llm.Request, headers map[string]string, out *anthropicResponse) (anthropicRequest, error) {
	plain := req
	plain.ThinkingBudget = 0
	body := c.buildRequest(plain)
	err := postJSON(ctx, c.http, "anthropic", c.baseURL+"/v1/messages", headers, body, out)
	return body, err
}

// parseAnthropicContent joins the response's text blocks into the visible answer
// and its thinking blocks into the captured reasoning.
func parseAnthropicContent(out anthropicResponse) (text, reasoning string) {
	var tb, rb strings.Builder
	for _, block := range out.Content {
		switch block.Type {
		case "text":
			tb.WriteString(block.Text)
		case "thinking":
			rb.WriteString(block.Thinking)
		}
	}
	return tb.String(), rb.String()
}

// isThinkingError reports whether a 400 body indicates a thinking-related
// problem (unsupported thinking, an invalid budget, or a temperature conflict),
// so only those 400s trigger the retry without thinking.
func isThinkingError(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, "thinking") ||
		strings.Contains(b, "budget") ||
		strings.Contains(b, "temperature")
}

// buildRequest assembles the wire request, hoisting system messages and, when a
// thinking budget is set, enabling extended thinking. The messages API requires
// max_tokens > budget_tokens and forbids temperature while thinking is on, so
// with thinking enabled the budget is floored, max_tokens is grown past it, and
// temperature is left unset.
func (c *AnthropicClient) buildRequest(req llm.Request) anthropicRequest {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = anthropicDefaultMaxTokens
	}

	body := anthropicRequest{
		Model:       req.Model,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		Messages:    make([]anthropicMessage, 0, len(req.Messages)),
	}

	if req.ThinkingBudget > 0 {
		budget := req.ThinkingBudget
		if budget < anthropicMinThinkingBudget {
			budget = anthropicMinThinkingBudget
		}
		body.Thinking = &anthropicThinking{Type: "enabled", BudgetTokens: budget}
		// max_tokens must exceed the thinking budget and still leave room for the
		// visible answer.
		body.MaxTokens = budget + maxTokens
		// Extended thinking requires temperature unset.
		body.Temperature = nil
	}

	var systemParts []string
	for _, m := range req.Messages {
		if m.Role == llm.RoleSystem {
			systemParts = append(systemParts, m.Content)
			continue
		}
		body.Messages = append(body.Messages, anthropicMessage{Role: string(m.Role), Content: m.Content})
	}
	body.System = strings.Join(systemParts, "\n\n")
	return body
}
