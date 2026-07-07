package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
