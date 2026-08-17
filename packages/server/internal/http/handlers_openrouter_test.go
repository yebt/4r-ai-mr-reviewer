package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/openrouter"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// stubOpenRouter is an openRouterLister that returns a fixed catalog and counts
// how many times it was asked (to prove the handler doesn't refetch per call).
type stubOpenRouter struct {
	models []openrouter.Model
	calls  int
}

func (s *stubOpenRouter) ListModels(_ context.Context) ([]openrouter.Model, error) {
	s.calls++
	return s.models, nil
}

func TestListOpenRouterModelsSortsByID(t *testing.T) {
	stub := &stubOpenRouter{models: []openrouter.Model{
		{ID: "openai/gpt-4o", Name: "OpenAI: GPT-4o", ContextLength: 128000},
		{ID: "Anthropic/Claude", Name: "Anthropic: Claude", ContextLength: 200000},
		{ID: "meta/llama", Name: "Meta: Llama", ContextLength: 8192},
	}}

	var set skills.Set
	s := NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, set, nil, "", nil, false, nil)
	s.openrouter = stub
	srv := httptest.NewServer(s.Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openrouter/models")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var got []openRouterModelDTO
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Sorted alphabetically by id, case-insensitive: Anthropic, meta, openai.
	wantIDs := []string{"Anthropic/Claude", "meta/llama", "openai/gpt-4o"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d models, want %d", len(got), len(wantIDs))
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Fatalf("model[%d].id = %q, want %q (order: %+v)", i, got[i].ID, id, got)
		}
	}
	if got[1].ContextLength != 8192 {
		t.Fatalf("contextLength not mapped: %+v", got[1])
	}

	// A second request must reuse the same lister (the handler never refetches
	// per call — caching lives behind the lister). The stub proves each request
	// makes exactly one lister call.
	resp2, err := http.Get(srv.URL + "/openrouter/models")
	if err != nil {
		t.Fatalf("GET 2: %v", err)
	}
	resp2.Body.Close()
	if stub.calls != 2 {
		t.Fatalf("lister calls = %d, want 2 (one per request)", stub.calls)
	}
}
