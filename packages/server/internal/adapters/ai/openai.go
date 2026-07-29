package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

// defaultOpenAIBaseURL is used when a provider config leaves BaseURL empty.
const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIClient talks to any OpenAI-compatible /chat/completions endpoint.
type OpenAIClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewOpenAIClient builds a client. An empty baseURL falls back to OpenAI's.
func NewOpenAIClient(baseURL, apiKey string) *OpenAIClient {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}
}

var _ llm.Client = (*OpenAIClient)(nil)

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openaiResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			// Some OpenAI-compatible providers expose the model's reasoning as a
			// sibling of content, under one of these two keys.
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete sends a chat completion request.
func (c *OpenAIClient) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	body := openaiRequest{
		Model:       req.Model,
		Messages:    make([]openaiMessage, 0, len(req.Messages)),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, openaiMessage{Role: string(m.Role), Content: m.Content})
	}

	var out openaiResponse
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}
	if err := postJSON(ctx, c.http, "openai-compat", c.baseURL+"/chat/completions", headers, body, &out); err != nil {
		return llm.Response{}, err
	}
	if len(out.Choices) == 0 {
		return llm.Response{}, fmt.Errorf("ai: openai-compat returned no choices")
	}
	msg := out.Choices[0].Message
	return llm.Response{
		Content:      msg.Content,
		InputTokens:  out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		Model:        out.Model,
		Reasoning:    firstNonEmpty(msg.ReasoningContent, msg.Reasoning),
	}, nil
}

// firstNonEmpty returns the first non-empty string, or "" if all are empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
