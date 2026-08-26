package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

// sseEvent formats one server-sent event (with an optional event name) plus the
// terminating blank line.
func sseEvent(event, data string) string {
	var b strings.Builder
	if event != "" {
		b.WriteString("event: ")
		b.WriteString(event)
		b.WriteByte('\n')
	}
	b.WriteString("data: ")
	b.WriteString(data)
	b.WriteString("\n\n")
	return b.String()
}

// writeSSE sets the SSE content type and writes each chunk, flushing between them
// so the client sees them incrementally.
func writeSSE(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	f, _ := w.(http.Flusher)
	for _, c := range chunks {
		io.WriteString(w, c)
		if f != nil {
			f.Flush()
		}
	}
}

// fastIdle shrinks the stream idle timeout for a test, restoring it after.
func fastIdle(t *testing.T, d time.Duration) {
	old := streamIdleTimeout
	streamIdleTimeout = d
	t.Cleanup(func() { streamIdleTimeout = old })
}

func TestAnthropicStreamComplete(t *testing.T) {
	var gotReq anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		writeSSE(w,
			sseEvent("message_start", `{"type":"message_start","message":{"model":"claude","usage":{"input_tokens":20,"output_tokens":1}}}`),
			sseEvent("content_block_delta", `{"type":"content_block_delta","delta":{"type":"text_delta","text":"part one "}}`),
			sseEvent("content_block_delta", `{"type":"content_block_delta","delta":{"type":"text_delta","text":"part two"}}`),
			sseEvent("message_delta", `{"type":"message_delta","usage":{"output_tokens":5}}`),
			sseEvent("message_stop", `{"type":"message_stop"}`),
		)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k") // streaming default
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:    "claude",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "review this"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !gotReq.Stream {
		t.Fatal("request must set stream:true")
	}
	if resp.Content != "part one part two" {
		t.Fatalf("content = %q, want concatenated text deltas", resp.Content)
	}
	if resp.InputTokens != 20 || resp.OutputTokens != 5 {
		t.Fatalf("tokens = %d/%d, want 20/5", resp.InputTokens, resp.OutputTokens)
	}
	if resp.Model != "claude" {
		t.Fatalf("model = %q, want claude", resp.Model)
	}
}

func TestAnthropicStreamThinking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			sseEvent("message_start", `{"message":{"model":"claude","usage":{"input_tokens":10}}}`),
			sseEvent("content_block_delta", `{"delta":{"type":"thinking_delta","thinking":"let me reason "}}`),
			sseEvent("content_block_delta", `{"delta":{"type":"thinking_delta","thinking":"further"}}`),
			sseEvent("content_block_delta", `{"delta":{"type":"text_delta","text":"final answer"}}`),
			sseEvent("message_delta", `{"usage":{"output_tokens":8}}`),
			sseEvent("message_stop", `{}`),
		)
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
	if resp.Reasoning != "let me reason further" {
		t.Fatalf("reasoning = %q, want joined thinking deltas", resp.Reasoning)
	}
	if resp.Content != "final answer" {
		t.Fatalf("content = %q, want final answer", resp.Content)
	}
}

// TestAnthropicStreamThinkingFallbackOn400 proves the thinking-off retry still
// fires over the streaming transport: a 400 thinking error before the stream
// starts is retried once without thinking, over SSE again.
func TestAnthropicStreamThinkingFallbackOn400(t *testing.T) {
	fastRetries(t)
	var calls int32
	var secondHadThinking bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		var req anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if n == 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"message":"thinking not supported"}}`)
			return
		}
		secondHadThinking = req.Thinking != nil
		writeSSE(w,
			sseEvent("content_block_delta", `{"delta":{"type":"text_delta","text":"recovered"}}`),
			sseEvent("message_stop", `{}`),
		)
	}))
	defer srv.Close()

	c := NewAnthropicClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:          "claude",
		ThinkingBudget: 2000,
		Messages:       []llm.Message{{Role: llm.RoleUser, Content: "review"}},
	})
	if err != nil {
		t.Fatalf("Complete after fallback: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (thinking 400 retried once without thinking)", got)
	}
	if secondHadThinking {
		t.Fatal("retry must not carry thinking")
	}
	if resp.Content != "recovered" {
		t.Fatalf("content = %q, want recovered", resp.Content)
	}
}

func TestOpenAIStreamComplete(t *testing.T) {
	var gotReq openaiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		writeSSE(w,
			sseEvent("", `{"model":"llama","choices":[{"delta":{"content":"looks "}}]}`),
			sseEvent("", `{"choices":[{"delta":{"content":"good"}}]}`),
			sseEvent("", `{"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":3}}`),
			sseEvent("", `[DONE]`),
		)
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k") // streaming default
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:    "llama",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !gotReq.Stream || gotReq.StreamOptions == nil || !gotReq.StreamOptions.IncludeUsage {
		t.Fatalf("request must set stream + stream_options.include_usage: %+v", gotReq)
	}
	if resp.Content != "looks good" {
		t.Fatalf("content = %q, want looks good", resp.Content)
	}
	if resp.InputTokens != 12 || resp.OutputTokens != 3 {
		t.Fatalf("tokens = %d/%d, want 12/3", resp.InputTokens, resp.OutputTokens)
	}
	if resp.Model != "llama" {
		t.Fatalf("model = %q, want llama", resp.Model)
	}
}

func TestOpenAIStreamReasoning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w,
			sseEvent("", `{"choices":[{"delta":{"reasoning_content":"deep "}}]}`),
			sseEvent("", `{"choices":[{"delta":{"reasoning_content":"thought"}}]}`),
			sseEvent("", `{"choices":[{"delta":{"content":"ok"}}]}`),
			sseEvent("", `[DONE]`),
		)
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{Model: "m"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Reasoning != "deep thought" {
		t.Fatalf("reasoning = %q, want deep thought", resp.Reasoning)
	}
	if resp.Content != "ok" {
		t.Fatalf("content = %q, want ok", resp.Content)
	}
}

// TestOpenAIStreamRetriesOn429 proves the streaming path applies the same 429
// retry policy to the initial response as the buffered path.
func TestOpenAIStreamRetriesOn429(t *testing.T) {
	fastRetries(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < maxAttempts {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writeSSE(w,
			sseEvent("", `{"choices":[{"delta":{"content":"ok"}}]}`),
			sseEvent("", `[DONE]`),
		)
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{Model: "m"})
	if err != nil {
		t.Fatalf("Complete after retries: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("content = %q, want ok", resp.Content)
	}
	if got := atomic.LoadInt32(&calls); got != maxAttempts {
		t.Fatalf("calls = %d, want %d", got, maxAttempts)
	}
}

// TestStreamIdleTimeout proves a stalled stream (no bytes after the first event)
// is aborted by the idle timeout rather than hanging until a total cap.
func TestStreamIdleTimeout(t *testing.T) {
	fastIdle(t, 50*time.Millisecond)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSSE(w, sseEvent("", `{"choices":[{"delta":{"content":"start"}}]}`))
		// Stall: never send another byte. Wait for the client to disconnect (its
		// idle timeout cancels the request) or give up after a bounded wait.
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k")
	start := time.Now()
	_, err := c.Complete(context.Background(), llm.Request{Model: "m"})
	if err == nil {
		t.Fatal("expected an idle timeout error")
	}
	if !strings.Contains(err.Error(), "idle timeout") {
		t.Fatalf("error = %v, want an idle timeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("idle timeout took %s, want ~50ms (it should not wait for the total cap)", elapsed)
	}
}
