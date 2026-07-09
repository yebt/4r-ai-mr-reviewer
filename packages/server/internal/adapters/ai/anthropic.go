package ai

import (
	"context"
	"fmt"
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

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete sends a messages request. System-role messages are hoisted into the
// top-level system field, as the Anthropic API requires.
func (c *AnthropicClient) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
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
	var systemParts []string
	for _, m := range req.Messages {
		if m.Role == llm.RoleSystem {
			systemParts = append(systemParts, m.Content)
			continue
		}
		body.Messages = append(body.Messages, anthropicMessage{Role: string(m.Role), Content: m.Content})
	}
	body.System = strings.Join(systemParts, "\n\n")

	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": anthropicVersion,
	}

	var out anthropicResponse
	if err := postJSON(ctx, c.http, "anthropic", c.baseURL+"/v1/messages", headers, body, &out); err != nil {
		return llm.Response{}, err
	}

	var text strings.Builder
	for _, block := range out.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return llm.Response{}, fmt.Errorf("ai: anthropic returned no text content")
	}
	return llm.Response{
		Content:      text.String(),
		InputTokens:  out.Usage.InputTokens,
		OutputTokens: out.Usage.OutputTokens,
		Model:        out.Model,
	}, nil
}
