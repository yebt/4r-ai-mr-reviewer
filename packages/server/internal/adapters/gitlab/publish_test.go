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

func TestApproveMergeRequest(t *testing.T) {
	var gotPath, gotMethod, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotToken = r.URL.Path, r.Method, r.Header.Get("PRIVATE-TOKEN")
		if !strings.HasSuffix(r.URL.Path, "/approve") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if err := c.ApproveMergeRequest(context.Background(), "g/p", 7); err != nil {
		t.Fatalf("ApproveMergeRequest: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/merge_requests/7/approve") {
		t.Errorf("path = %q, want .../merge_requests/7/approve", gotPath)
	}
	if gotToken != "tok" {
		t.Errorf("token = %q, want tok", gotToken)
	}
}

// TestApproveMergeRequestForbiddenIsError: a 403 is a real permission failure and
// must surface, not be swallowed as idempotency.
func TestApproveMergeRequestForbiddenIsError(t *testing.T) {
	srv := cannedServer(t, http.StatusForbidden, `{"message":"403 Forbidden"}`)
	c := NewClient(srv.URL, "tok")
	err := c.ApproveMergeRequest(context.Background(), "g/p", 7)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("403 must surface as APIError, got %v", err)
	}
}

// TestApproveMergeRequestAlreadyApprovedIsSuccess: an already-approved MR (401
// with an "already approved" body) is swallowed so a routine re-run does not fail.
func TestApproveMergeRequestAlreadyApprovedIsSuccess(t *testing.T) {
	srv := cannedServer(t, http.StatusUnauthorized, `{"message":"You have already approved this merge request"}`)
	c := NewClient(srv.URL, "tok")
	if err := c.ApproveMergeRequest(context.Background(), "g/p", 7); err != nil {
		t.Fatalf("already-approved 401 should be nil, got %v", err)
	}
}

// cannedServer answers every request with a fixed status and body.
func cannedServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestAwardEmojiAlreadyAwardedIsSuccess: a duplicate award (404 with an "already
// awarded" body) is swallowed so a routine re-run does not fail on a reaction it
// set on a previous pass.
func TestAwardEmojiAlreadyAwardedIsSuccess(t *testing.T) {
	srv := cannedServer(t, http.StatusNotFound, `{"message":"Emoji has already been awarded"}`)
	c := NewClient(srv.URL, "tok")
	if err := c.AwardEmoji(context.Background(), "g/p", 7, "thumbsup"); err != nil {
		t.Fatalf("already-awarded 404 should be nil, got %v", err)
	}
}

// TestAwardEmojiAlready400IsSuccess: some GitLab versions answer a duplicate
// award with 400 "has already been taken" (not 404/409). That must still be
// treated as an already-present reaction (success), so the react step does not
// block when a previous pass already reacted.
func TestAwardEmojiAlready400IsSuccess(t *testing.T) {
	srv := cannedServer(t, http.StatusBadRequest, `{"message":{"name":["has already been taken"]}}`)
	c := NewClient(srv.URL, "tok")
	if err := c.AwardEmoji(context.Background(), "g/p", 7, "thumbsup"); err != nil {
		t.Fatalf("already-taken 400 should be nil, got %v", err)
	}
}

// TestAwardEmojiBadRequestWithoutAlreadyIsError: a 400 that is NOT about a
// duplicate must surface, not be swallowed as idempotency.
func TestAwardEmojiBadRequestWithoutAlreadyIsError(t *testing.T) {
	srv := cannedServer(t, http.StatusBadRequest, `{"message":"400 Bad Request"}`)
	c := NewClient(srv.URL, "tok")
	err := c.AwardEmoji(context.Background(), "g/p", 7, "thumbsup")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("400 without 'already' must surface as APIError, got %v", err)
	}
}

// TestAwardEmojiForbiddenIsError: a 403 is a real failure (no token permission)
// and must NOT be swallowed as if it were a duplicate award.
func TestAwardEmojiForbiddenIsError(t *testing.T) {
	srv := cannedServer(t, http.StatusForbidden, `{"message":"403 Forbidden"}`)
	c := NewClient(srv.URL, "tok")
	err := c.AwardEmoji(context.Background(), "g/p", 7, "thumbsup")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("403 must surface as APIError, got %v", err)
	}
}

// TestAwardEmojiNotFoundWithoutAlreadyIsError: a 404 that is NOT about a duplicate
// (e.g. the MR does not exist) must surface, not be mistaken for idempotency.
func TestAwardEmojiNotFoundWithoutAlreadyIsError(t *testing.T) {
	srv := cannedServer(t, http.StatusNotFound, `{"message":"404 Merge Request Not Found"}`)
	c := NewClient(srv.URL, "tok")
	err := c.AwardEmoji(context.Background(), "g/p", 7, "thumbsup")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		t.Fatalf("404 without 'already' must surface as APIError, got %v", err)
	}
}

// TestDoFormEmptyBodyWithOutIsSuccess: a 2xx with an empty body is success even
// when a response object was requested — it must not be a JSON decode error.
func TestDoFormEmptyBodyWithOutIsSuccess(t *testing.T) {
	srv := cannedServer(t, http.StatusCreated, "")
	c := NewClient(srv.URL, "tok")
	if _, err := c.CreateMergeRequest(context.Background(), "g/p", "feat", "main", "title", "desc"); err != nil {
		t.Fatalf("201 with empty body should decode as success, got %v", err)
	}
}
