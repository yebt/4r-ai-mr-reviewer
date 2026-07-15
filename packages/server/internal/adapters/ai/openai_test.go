package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
)

func TestOpenAIComplete(t *testing.T) {
	var gotReq openaiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("Authorization = %q, want Bearer k", got)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"llama","choices":[{"message":{"content":"looks good"}}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`)
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k")
	resp, err := c.Complete(context.Background(), llm.Request{
		Model:    "llama",
		Messages: []llm.Message{{Role: llm.RoleSystem, Content: "sys"}, {Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "looks good" || resp.InputTokens != 12 || resp.OutputTokens != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if gotReq.Model != "llama" || len(gotReq.Messages) != 2 {
		t.Fatalf("unexpected request forwarded: %+v", gotReq)
	}
	if gotReq.Temperature != nil {
		t.Errorf("temperature should be omitted when unset, got %v", *gotReq.Temperature)
	}
}

func TestOpenAISendsTemperatureWhenSet(t *testing.T) {
	var gotReq openaiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		fmt.Fprint(w, `{"model":"m","choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()

	temp := 0.3
	c := NewOpenAIClient(srv.URL, "k")
	if _, err := c.Complete(context.Background(), llm.Request{Model: "m", Temperature: &temp}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if gotReq.Temperature == nil || *gotReq.Temperature != 0.3 {
		t.Fatalf("temperature not sent: %v", gotReq.Temperature)
	}
}

func TestOpenAINoChoicesIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"model":"x","choices":[]}`)
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k")
	if _, err := c.Complete(context.Background(), llm.Request{Model: "x"}); err == nil {
		t.Fatal("expected error on empty choices")
	}
}

func TestOpenAIAPIError(t *testing.T) {
	fastRetries(t)

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `rate limited`)
	}))
	defer srv.Close()

	c := NewOpenAIClient(srv.URL, "k")
	_, err := c.Complete(context.Background(), llm.Request{Model: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
		t.Fatalf("expected APIError 429, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != maxAttempts {
		t.Fatalf("calls = %d, want %d (429 should be retried)", got, maxAttempts)
	}
}

func TestOpenAIRetriesThenSucceeds(t *testing.T) {
	fastRetries(t)

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < maxAttempts {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
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

// fastRetries shrinks the backoff so retry tests run instantly, restoring it after.
func fastRetries(t *testing.T) {
	old := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = old })
}
