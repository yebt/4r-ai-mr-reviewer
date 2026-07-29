package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListOpenMergeRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/merge_requests") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.URL.Query().Get("state"); got != "opened" {
			t.Errorf("state = %q, want opened", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"iid":7,"title":"Add feature","state":"opened","source_branch":"feat","target_branch":"main","author":{"username":"yahir","name":"Yahir"}}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	mrs, err := c.ListOpenMergeRequests(context.Background(), "group/project")
	if err != nil {
		t.Fatalf("ListOpenMergeRequests: %v", err)
	}
	if len(mrs) != 1 {
		t.Fatalf("len = %d, want 1", len(mrs))
	}
	if mrs[0].IID != 7 || mrs[0].Author.Username != "yahir" {
		t.Fatalf("unexpected mr: %+v", mrs[0])
	}
}

func TestMergeRequestChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/changes") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"iid":7,"title":"Add feature","sha":"abc","changes":[{"old_path":"a.go","new_path":"a.go","diff":"@@ -1 +1 @@","new_file":false}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	ch, err := c.MergeRequestChanges(context.Background(), "group/project", 7)
	if err != nil {
		t.Fatalf("MergeRequestChanges: %v", err)
	}
	if ch.IID != 7 || ch.SHA != "abc" {
		t.Fatalf("unexpected mr fields: %+v", ch.MergeRequest)
	}
	if len(ch.Files) != 1 || ch.Files[0].NewPath != "a.go" {
		t.Fatalf("unexpected changes: %+v", ch.Files)
	}
}

func TestAPIErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.ListOpenMergeRequests(context.Background(), "p")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", apiErr.Status)
	}
}

func TestCompareRefs(t *testing.T) {
	var gotFrom, gotTo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/repository/compare") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotFrom = r.URL.Query().Get("from")
		gotTo = r.URL.Query().Get("to")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"commits":[{"id":"a1","title":"feat: one","message":"feat: one\n\nbody"},{"id":"b2","title":"fix: two","message":"fix: two"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	cmp, err := c.CompareRefs(context.Background(), "group/project", "v1.0.0", "main")
	if err != nil {
		t.Fatalf("CompareRefs: %v", err)
	}
	if gotFrom != "v1.0.0" || gotTo != "main" {
		t.Fatalf("query from/to = %q/%q, want v1.0.0/main", gotFrom, gotTo)
	}
	if len(cmp.Commits) != 2 {
		t.Fatalf("len = %d, want 2", len(cmp.Commits))
	}
	if cmp.Commits[0].ID != "a1" || cmp.Commits[0].Title != "feat: one" {
		t.Fatalf("commit[0] = %+v", cmp.Commits[0])
	}
	if cmp.Commits[1].Title != "fix: two" {
		t.Fatalf("commit[1] = %+v", cmp.Commits[1])
	}
}

func TestMergeRequestCommits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/merge_requests/7/commits") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"c1","title":"feat: newest","message":"feat: newest"},{"id":"c2","title":"fix: older","message":"fix: older"}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	commits, err := c.MergeRequestCommits(context.Background(), "group/project", 7)
	if err != nil {
		t.Fatalf("MergeRequestCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("len = %d, want 2", len(commits))
	}
	if commits[0].ID != "c1" || commits[0].Title != "feat: newest" {
		t.Fatalf("commit[0] = %+v", commits[0])
	}
	if commits[1].Title != "fix: older" {
		t.Fatalf("commit[1] = %+v", commits[1])
	}
}

func TestMergeRequestMergeStatusFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"iid":7,"has_conflicts":true,"merge_status":"cannot_be_merged","detailed_merge_status":"conflict"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	mr, err := c.GetMergeRequest(context.Background(), "g/p", 7)
	if err != nil {
		t.Fatalf("GetMergeRequest: %v", err)
	}
	if !mr.HasConflicts {
		t.Errorf("HasConflicts = false, want true")
	}
	if mr.MergeStatus != "cannot_be_merged" {
		t.Errorf("MergeStatus = %q, want cannot_be_merged", mr.MergeStatus)
	}
	if mr.DetailedMergeStatus != "conflict" {
		t.Errorf("DetailedMergeStatus = %q, want conflict", mr.DetailedMergeStatus)
	}
}

func TestInjectUsername(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://gitlab.com/g/p.git", "https://oauth2@gitlab.com/g/p.git"},
		{"http://host/g/p.git", "http://oauth2@host/g/p.git"},
		{"file:///tmp/repo", "file:///tmp/repo"}, // non-http untouched
	}
	for _, tc := range cases {
		got, err := injectUsername(tc.in, "oauth2")
		if err != nil {
			t.Fatalf("injectUsername(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("injectUsername(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
