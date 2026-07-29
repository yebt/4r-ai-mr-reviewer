package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

func TestAnthropicComplete(t *testing.T) {
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "k" {
			t.Errorf("x-api-key = %q, want k", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Errorf("anthropic-version = %q, want %q", got, anthropicVersion)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/messages") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"claude","content":[{"type":"text","text":"part one "},{"type":"text","text":"part two"}],"usage":{"input_tokens":20,"output_tokens":5}}`)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:    "claude",
		Messages: []llm.Message{{Role: llm.RoleSystem, Content: "you are a reviewer"}, {Role: llm.RoleUser, Content: "review this"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "part one part two" {
		t.Fatalf("content = %q, want concatenated blocks", resp.Content)
	}
	if resp.InputTokens != 20 || resp.OutputTokens != 5 {
		t.Fatalf("tokens = %d/%d, want 20/5", resp.InputTokens, resp.OutputTokens)
	}

	// System message must be hoisted out of messages into the system field.
	if gotReq.System != "you are a reviewer" {
		t.Fatalf("system = %q, want hoisted system prompt", gotReq.System)
	}
	if len(gotReq.Messages) != 1 || gotReq.Messages[0].Role != "user" {
		t.Fatalf("messages should exclude the system role: %+v", gotReq.Messages)
	}
	// max_tokens must default because the request left it 0.
	if gotReq.MaxTokens != anthropicDefaultMaxTokens {
		t.Fatalf("max_tokens = %d, want default %d", gotReq.MaxTokens, anthropicDefaultMaxTokens)
	}
}

func TestAnthropicThinkingRequestAndParse(t *testing.T) {
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"claude","content":[{"type":"thinking","thinking":"let me reason "},{"type":"thinking","thinking":"further"},{"type":"text","text":"final answer"}],"usage":{"input_tokens":10,"output_tokens":8}}`)
	}))
	defer srv.Close()

	temp := 0.4
	c := NewAnthropicClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:          "claude",
		Temperature:    &temp,
		ThinkingBudget: 2000,
		Messages:       []llm.Message{{Role: llm.RoleUser, Content: "review"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Request must enable thinking with the requested budget and omit temperature.
	if gotReq.Thinking == nil || gotReq.Thinking.Type != "enabled" || gotReq.Thinking.BudgetTokens != 2000 {
		t.Fatalf("thinking not set correctly: %+v", gotReq.Thinking)
	}
	if gotReq.Temperature != nil {
		t.Fatalf("temperature must be omitted with thinking, got %v", *gotReq.Temperature)
	}
	if gotReq.MaxTokens <= gotReq.Thinking.BudgetTokens {
		t.Fatalf("max_tokens = %d must exceed thinking budget %d", gotReq.MaxTokens, gotReq.Thinking.BudgetTokens)
	}

	// Response: thinking blocks joined into Reasoning, text into Content.
	if resp.Reasoning != "let me reason further" {
		t.Fatalf("reasoning = %q, want joined thinking blocks", resp.Reasoning)
	}
	if resp.Content != "final answer" {
		t.Fatalf("content = %q, want text block", resp.Content)
	}
}

func TestAnthropicThinkingFloorsBudget(t *testing.T) {
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		fmt.Fprint(w, `{"model":"claude","content":[{"type":"text","text":"ok"}]}`)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k")
	if _, err := c.Complete(context.Background(), llm.Request{Model: "claude", ThinkingBudget: 10}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if gotReq.Thinking == nil || gotReq.Thinking.BudgetTokens != anthropicMinThinkingBudget {
		t.Fatalf("budget = %+v, want floored to %d", gotReq.Thinking, anthropicMinThinkingBudget)
	}
}

func TestAnthropicThinkingFallbackOn400(t *testing.T) {
	var calls int32
	var secondHadThinking bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		var req anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if n == 1 {
			// First call carries thinking: reject it as unsupported.
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"thinking not supported"}`)
			return
		}
		secondHadThinking = req.Thinking != nil
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"claude","content":[{"type":"text","text":"recovered"}],"usage":{"input_tokens":3,"output_tokens":2}}`)
	}))
	defer srv.Close()

	temp := 0.3
	c := NewAnthropicClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:          "claude",
		Temperature:    &temp,
		ThinkingBudget: 2000,
		Messages:       []llm.Message{{Role: llm.RoleUser, Content: "review"}},
	})
	if err != nil {
		t.Fatalf("Complete after fallback: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (thinking request retried once without thinking)", got)
	}
	if secondHadThinking {
		t.Fatal("retry must not carry thinking")
	}
	if resp.Content != "recovered" {
		t.Fatalf("content = %q, want recovered", resp.Content)
	}
}

// TestAnthropicThinkingOnlyRetriesWithoutThinking covers a valid 200 that comes
// back with only a thinking block and no text (the model exhausted max_tokens
// while reasoning): the adapter must retry once without thinking and return the
// text from that second call, rather than failing the pass.
func TestAnthropicThinkingOnlyRetriesWithoutThinking(t *testing.T) {
	var calls int32
	var secondHadThinking bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		var req anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// Thinking-only 200: no text block at all.
			fmt.Fprint(w, `{"model":"claude","content":[{"type":"thinking","thinking":"still reasoning"}],"usage":{"input_tokens":10,"output_tokens":9}}`)
			return
		}
		secondHadThinking = req.Thinking != nil
		fmt.Fprint(w, `{"model":"claude","content":[{"type":"text","text":"the answer"}],"usage":{"input_tokens":4,"output_tokens":3}}`)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:          "claude",
		ThinkingBudget: 2000,
		Messages:       []llm.Message{{Role: llm.RoleUser, Content: "review"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (thinking-only 200 retried once without thinking)", got)
	}
	if secondHadThinking {
		t.Fatal("retry must not carry thinking")
	}
	if resp.Content != "the answer" {
		t.Fatalf("content = %q, want the answer", resp.Content)
	}
}

// TestAnthropicUnrelated400DoesNotRetry asserts the 400 fallback only fires for
// thinking-related errors: an unrelated 400 (bad model) with thinking on must
// surface immediately, without a second call.
func TestAnthropicUnrelated400DoesNotRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"invalid model"}}`)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k")
	_, err := c.Complete(context.Background(), llm.Request{
		Model:          "claude",
		ThinkingBudget: 2000,
		Messages:       []llm.Message{{Role: llm.RoleUser, Content: "review"}},
	})
	if err == nil {
		t.Fatal("expected the original 400 to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (an unrelated 400 must not retry without thinking)", got)
	}
}

func TestAnthropicNoTextIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"model":"claude","content":[]}`)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k")
	if _, err := c.Complete(context.Background(), llm.Request{Model: "claude"}); err == nil {
		t.Fatal("expected error when no text content is returned")
	}
}
