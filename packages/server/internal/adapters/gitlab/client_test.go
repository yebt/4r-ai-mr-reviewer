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
