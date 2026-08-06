package openrouter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// catalogStub serves a small OpenRouter catalog and counts upstream hits.
func catalogStub(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[
			{"id":"openai/gpt-4o","name":"OpenAI: GPT-4o","context_length":128000},
			{"id":"anthropic/claude-3.5","name":"Anthropic: Claude 3.5","context_length":200000}
		]}`)
	}))
}

func TestListModelsParses(t *testing.T) {
	var hits int32
	srv := catalogStub(t, &hits)
	defer srv.Close()

	c := &Client{http: srv.Client(), url: srv.URL}
	models, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0].ID != "openai/gpt-4o" || models[0].Name != "OpenAI: GPT-4o" || models[0].ContextLength != 128000 {
		t.Fatalf("unexpected first model: %+v", models[0])
	}
}

func TestCachedClientServesSecondCallWithoutUpstreamHit(t *testing.T) {
	var hits int32
	srv := catalogStub(t, &hits)
	defer srv.Close()

	c := &Client{http: srv.Client(), url: srv.URL}
	cached := &CachedClient{fetch: c.ListModels, ttl: time.Hour}

	first, err := cached.ListModels(context.Background())
	if err != nil {
		t.Fatalf("first ListModels: %v", err)
	}
	second, err := cached.ListModels(context.Background())
	if err != nil {
		t.Fatalf("second ListModels: %v", err)
	}
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("cached lengths: first=%d second=%d, want 2/2", len(first), len(second))
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("upstream hits = %d, want 1 (cache should serve the second call)", got)
	}
}

func TestCachedClientServesStaleOnError(t *testing.T) {
	var fail bool
	call := int32(0)
	cached := &CachedClient{
		ttl: 0, // always expired, forcing a refetch on every call
		fetch: func(_ context.Context) ([]Model, error) {
			atomic.AddInt32(&call, 1)
			if fail {
				return nil, fmt.Errorf("boom")
			}
			return []Model{{ID: "a/b"}}, nil
		},
	}

	if _, err := cached.ListModels(context.Background()); err != nil {
		t.Fatalf("warm-up: %v", err)
	}
	fail = true
	got, err := cached.ListModels(context.Background())
	if err != nil {
		t.Fatalf("expected stale fallback, got error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a/b" {
		t.Fatalf("stale fallback returned %+v, want [{a/b}]", got)
	}
}
