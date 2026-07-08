package gitlab

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCreateNote(t *testing.T) {
	var gotBody, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/notes") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		_ = r.ParseForm()
		gotBody = r.FormValue("body")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if err := c.CreateNote(context.Background(), "g/p", 7, "summary body"); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if gotBody != "summary body" || gotToken != "tok" {
		t.Fatalf("note not posted correctly: body=%q token=%q", gotBody, gotToken)
	}
}

func TestCreateInlineDiscussion(t *testing.T) {
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/discussions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = r.ParseForm()
		form = r.Form
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	pos := Position{BaseSHA: "b", StartSHA: "s", HeadSHA: "h", NewPath: "auth.go", NewLine: 42}
	if err := c.CreateInlineDiscussion(context.Background(), "g/p", 7, "finding body", pos); err != nil {
		t.Fatalf("CreateInlineDiscussion: %v", err)
	}
	if form.Get("body") != "finding body" {
		t.Errorf("body = %q", form.Get("body"))
	}
	if form.Get("position[position_type]") != "text" {
		t.Errorf("position_type = %q, want text", form.Get("position[position_type]"))
	}
	if form.Get("position[new_path]") != "auth.go" || form.Get("position[new_line]") != "42" {
		t.Errorf("new_path/new_line wrong: %q/%q", form.Get("position[new_path]"), form.Get("position[new_line]"))
	}
	if form.Get("position[head_sha]") != "h" || form.Get("position[base_sha]") != "b" {
		t.Errorf("shas wrong: head=%q base=%q", form.Get("position[head_sha]"), form.Get("position[base_sha]"))
	}
}

func TestPublishReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	err := c.CreateNote(context.Background(), "g/p", 7, "x")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("expected APIError 400, got %v", err)
	}
}
