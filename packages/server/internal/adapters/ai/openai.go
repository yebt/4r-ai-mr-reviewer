package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

// defaultOpenAIBaseURL is used when a provider config leaves BaseURL empty.
const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIClient talks to any OpenAI-compatible /chat/completions endpoint.
type OpenAIClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	// stream sends requests as SSE and consumes them incrementally, so a long
	// generation is bounded by an idle timeout rather than a blunt total cap.
	// Defaults on; AIR_AI_STREAM=false falls back to the buffered path.
	stream bool
}

// NewOpenAIClient builds a client. An empty baseURL falls back to OpenAI's.
func NewOpenAIClient(baseURL, apiKey string) *OpenAIClient {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		// No total Timeout: streaming is idle-bounded; the buffered path applies its
		// own overall deadline (see postJSON).
		http:   &http.Client{},
		stream: streamDefault(),
	}
}

var _ llm.Client = (*OpenAIClient)(nil)

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openaiRequest struct {
	Model         string               `json:"model"`
	Messages      []openaiMessage      `json:"messages"`
	Temperature   *float64             `json:"temperature,omitempty"`
	MaxTokens     int                  `json:"max_tokens,omitempty"`
	Stream        bool                 `json:"stream,omitempty"`
	StreamOptions *openaiStreamOptions `json:"stream_options,omitempty"`
}

// openaiRespMessage is the assistant message. Some OpenAI-compatible providers
// expose the model's reasoning as a sibling of content, under one of these keys.
type openaiRespMessage struct {
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
	Reasoning        string `json:"reasoning"`
}

type openaiChoice struct {
	Message openaiRespMessage `json:"message"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type openaiResponse struct {
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
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
	if err := c.do(ctx, body, headers, &out); err != nil {
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

// do performs one request, decoding into out. It streams the response as SSE
// when streaming is enabled (bounded by an idle timeout), otherwise buffers it.
// Both paths fill the same openaiResponse.
func (c *OpenAIClient) do(ctx context.Context, body openaiRequest, headers map[string]string, out *openaiResponse) error {
	url := c.baseURL + "/chat/completions"
	if c.stream {
		body.Stream = true
		// Ask for token usage in the terminal chunk; without it a streamed response
		// carries no usage totals. Providers that ignore stream_options simply omit
		// usage, which degrades to zero counts rather than failing.
		body.StreamOptions = &openaiStreamOptions{IncludeUsage: true}
		return streamOpenAI(ctx, c.http, url, headers, body, out)
	}
	return postJSON(ctx, c.http, "openai-compat", url, headers, body, out)
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
