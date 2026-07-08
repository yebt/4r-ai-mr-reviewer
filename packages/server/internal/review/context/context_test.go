package reviewctx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
)

func TestFastStrategyBuildsDiffInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/changes") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"iid":7,"title":"Add login","description":"adds auth","changes":[{"old_path":"a.go","new_path":"a.go","diff":"@@ hunk @@"}]}`))
	}))
	defer srv.Close()

	strat := NewFastStrategy(gitlab.NewClient(srv.URL, "tok"))
	in, cleanup, err := strat.Build(context.Background(), Target{ProjectID: "g/p", MRIID: 7})
	defer cleanup()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if in.MRIID != 7 || in.Title != "Add login" || in.Description != "adds auth" {
		t.Fatalf("input metadata wrong: %+v", in)
	}
	if !strings.Contains(in.Diff, "a.go") || !strings.Contains(in.Diff, "@@ hunk @@") {
		t.Fatalf("diff not rendered: %q", in.Diff)
	}
}

func TestFastStrategyPropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	strat := NewFastStrategy(gitlab.NewClient(srv.URL, "tok"))
	if _, cleanup, err := strat.Build(context.Background(), Target{ProjectID: "g/p", MRIID: 1}); err == nil {
		cleanup()
		t.Fatal("expected error from failing GitLab API")
	}
}
