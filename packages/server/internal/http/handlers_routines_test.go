package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakePreflightGitLab serves the endpoints the preflight probes with a fixed,
// well-formed project so the /preflight wire contract can be exercised.
func fakePreflightGitLab(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal_access_tokens/self"):
			_ = json.NewEncoder(w).Encode(map[string]any{"active": true, "scopes": []string{"api"}})
		case strings.HasSuffix(r.URL.Path, "/protected_branches"):
			_ = json.NewEncoder(w).Encode([]any{})
		case strings.HasSuffix(r.URL.Path, "/protected_tags"):
			_ = json.NewEncoder(w).Encode([]any{})
		case strings.Contains(r.URL.Path, "/projects/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"default_branch": "main",
				"permissions":    map[string]any{"project_access": map[string]any{"access_level": 40}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRepoPreflightOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	gl := fakePreflightGitLab(t)

	// The account's base URL points at the fake GitLab so the preflight resolves.
	acctResp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "a", "baseUrl": gl.URL, "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	repoResp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.test/g/p", "accountId": acct.ID})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo status = %d, want 201", repoResp.StatusCode)
	}
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)

	resp, err := http.Get(srv.URL + "/repos/" + repoObj.ID + "/preflight")
	if err != nil {
		t.Fatalf("GET preflight: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preflight status = %d, want 200", resp.StatusCode)
	}
	var pf struct {
		TokenScopes     []string `json:"tokenScopes"`
		ScopesKnown     bool     `json:"scopesKnown"`
		AccessLevelName string   `json:"accessLevelName"`
		Checks          []struct {
			Capability string `json:"capability"`
			Status     string `json:"status"`
		} `json:"checks"`
	}
	decodeBody(t, resp, &pf)

	if !containsString(pf.TokenScopes, "api") {
		t.Errorf("tokenScopes = %v, want to contain api", pf.TokenScopes)
	}
	if pf.AccessLevelName != "Maintainer" {
		t.Errorf("accessLevelName = %q, want Maintainer", pf.AccessLevelName)
	}
	if len(pf.Checks) != 5 {
		t.Fatalf("checks len = %d, want 5", len(pf.Checks))
	}
	for _, c := range pf.Checks {
		if c.Status == "" {
			t.Errorf("check %q has empty status", c.Capability)
		}
	}
}

func TestRepoPreflightUnknownRepoOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/repos/does-not-exist/preflight")
	if err != nil {
		t.Fatalf("GET preflight: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("preflight unknown repo status = %d, want 404", resp.StatusCode)
	}
}

// newRepoForRoutine creates an account and repo on srv and returns the repo id.
// No GitLab call happens at routine-creation time, so the account base URL can be
// arbitrary here.
func newRepoForRoutine(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	acctResp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "a", "baseUrl": "https://gitlab.com", "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	repoResp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": acct.ID})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo status = %d, want 201", repoResp.StatusCode)
	}
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)
	return repoObj.ID
}

func TestCreateApproveAndTagOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	repoID := newRepoForRoutine(t, srv)

	resp := postJSON(t, srv.URL+"/repos/"+repoID+"/routines/approve-and-tag", map[string]any{"mrIid": 7})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create routine status = %d, want 201", resp.StatusCode)
	}
	var run struct {
		ID     string `json:"id"`
		Kind   string `json:"kind"`
		Status string `json:"status"`
		MRIID  int    `json:"mrIid"`
		Steps  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"steps"`
		State json.RawMessage `json:"state"`
	}
	decodeBody(t, resp, &run)
	if run.ID == "" || run.Status != "pending" || run.Kind != "approve_and_tag" || run.MRIID != 7 {
		t.Fatalf("unexpected created run: %+v", run)
	}
	if len(run.Steps) != 3 || run.Steps[0].Name != "react" {
		t.Fatalf("unexpected steps: %+v", run.Steps)
	}
	if string(run.State) != "{}" {
		t.Fatalf("state = %s, want {}", run.State)
	}

	// It is retrievable by id.
	getResp, err := http.Get(srv.URL + "/routines/" + run.ID)
	if err != nil {
		t.Fatalf("GET routine: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get routine status = %d, want 200", getResp.StatusCode)
	}
	var got struct {
		ID string `json:"id"`
	}
	decodeBody(t, getResp, &got)
	if got.ID != run.ID {
		t.Fatalf("got id = %q, want %q", got.ID, run.ID)
	}

	// It shows up under the repo's routines.
	listResp, _ := http.Get(srv.URL + "/repos/" + repoID + "/routines")
	var list []map[string]any
	decodeBody(t, listResp, &list)
	if len(list) != 1 {
		t.Fatalf("repo routines len = %d, want 1", len(list))
	}
}

// fakeReleaseMRGitLab serves a single dev-flow MR so CreateRelease's up-front MR
// fetch (for the target-branch gate) resolves over HTTP.
func fakeReleaseMRGitLab(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/merge_requests/") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"iid": 7, "state": "opened",
				"source_branch": "feature", "target_branch": "development",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCreateReleaseOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	gl := fakeReleaseMRGitLab(t)

	acctResp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "a", "baseUrl": gl.URL, "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)
	repoResp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.test/g/p", "accountId": acct.ID})
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)

	resp := postJSON(t, srv.URL+"/repos/"+repoObj.ID+"/routines/release", map[string]any{"mrIid": 7})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create release status = %d, want 201", resp.StatusCode)
	}
	var run struct {
		Kind   string `json:"kind"`
		Status string `json:"status"`
		Steps  []struct {
			Name string `json:"name"`
		} `json:"steps"`
	}
	decodeBody(t, resp, &run)
	if run.Kind != "release" || run.Status != "pending" {
		t.Fatalf("unexpected created run: %+v", run)
	}
	if len(run.Steps) != 8 || run.Steps[0].Name != "verify" || run.Steps[4].Name != "confirm" {
		t.Fatalf("unexpected release steps: %+v", run.Steps)
	}
}

func TestCreateMainReleaseOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	// The main flow does NOT fetch an MR at creation, so an arbitrary account base
	// URL is fine here (no GitLab call happens until the worker runs).
	repoID := newRepoForRoutine(t, srv)

	resp := postJSON(t, srv.URL+"/repos/"+repoID+"/routines/release-main", map[string]any{"bump": "minor"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create main release status = %d, want 201", resp.StatusCode)
	}
	var run struct {
		Kind   string `json:"kind"`
		Status string `json:"status"`
		MRIID  int    `json:"mrIid"`
		Steps  []struct {
			Name string `json:"name"`
		} `json:"steps"`
	}
	decodeBody(t, resp, &run)
	if run.Kind != "release" || run.Status != "pending" {
		t.Fatalf("unexpected created run: %+v", run)
	}
	if run.MRIID != 0 {
		t.Errorf("mrIid = %d, want 0 (no MR at creation)", run.MRIID)
	}
	// The main flow now pauses on a confirm gate before merging: compute_tag,
	// create_mr, wait_pipeline, approve, react, confirm, merge, tag, notify.
	if len(run.Steps) != 9 || run.Steps[0].Name != "compute_tag" || run.Steps[1].Name != "create_mr" || run.Steps[5].Name != "confirm" {
		t.Fatalf("unexpected main-flow steps: %+v", run.Steps)
	}
}

func TestConfirmInvalidDecisionOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	// Invalid decision is rejected before the run is loaded, so any id yields 400.
	resp := postJSON(t, srv.URL+"/routines/any-id/confirm", map[string]any{"decision": "nope"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("confirm invalid decision status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestConfirmUnknownRunOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	// A valid decision on a run id that does not exist must surface as 404 (the
	// run-not-found path), not a blanket 400.
	resp := postJSON(t, srv.URL+"/routines/does-not-exist/confirm", map[string]any{"decision": "merge"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("confirm unknown run status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestResumeNonBlockedRoutineOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	repoID := newRepoForRoutine(t, srv)

	resp := postJSON(t, srv.URL+"/repos/"+repoID+"/routines/approve-and-tag", map[string]any{"mrIid": 7})
	var run struct {
		ID string `json:"id"`
	}
	decodeBody(t, resp, &run)

	// The run is pending, not blocked → resume is a 409 conflict.
	resumeResp := postJSON(t, srv.URL+"/routines/"+run.ID+"/resume", nil)
	if resumeResp.StatusCode != http.StatusConflict {
		t.Fatalf("resume non-blocked status = %d, want 409", resumeResp.StatusCode)
	}
	resumeResp.Body.Close()
}

func TestGetUnknownRoutineOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/routines/does-not-exist")
	if err != nil {
		t.Fatalf("GET routine: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown routine status = %d, want 404", resp.StatusCode)
	}
}
