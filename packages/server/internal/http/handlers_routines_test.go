package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/routines"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
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

// TestListRecentRoutinesOverHTTP verifies GET /routines returns runs newest
// first with a best-effort repoName on each item.
func TestListRecentRoutinesOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	repoID := newRepoForRoutine(t, srv)

	// Two runs on the same repo; the mrIid=8 run is created last, so it is newest.
	for _, iid := range []int{7, 8} {
		resp := postJSON(t, srv.URL+"/repos/"+repoID+"/routines/approve-and-tag", map[string]any{"mrIid": iid})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create routine (mr %d) status = %d, want 201", iid, resp.StatusCode)
		}
		resp.Body.Close()
		// Ensure distinct created_at timestamps so the newest-first ordering is
		// deterministic regardless of how fast the two creates run.
		time.Sleep(2 * time.Millisecond)
	}

	resp, err := http.Get(srv.URL + "/routines")
	if err != nil {
		t.Fatalf("GET routines: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list recent routines status = %d, want 200", resp.StatusCode)
	}
	var list []struct {
		MRIID    int    `json:"mrIid"`
		RepoID   string `json:"repoId"`
		RepoName string `json:"repoName"`
	}
	decodeBody(t, resp, &list)
	if len(list) != 2 {
		t.Fatalf("recent routines len = %d, want 2", len(list))
	}
	// Newest first: the mrIid=8 run precedes the mrIid=7 run.
	if list[0].MRIID != 8 || list[1].MRIID != 7 {
		t.Fatalf("expected newest first, got mr %d then mr %d", list[0].MRIID, list[1].MRIID)
	}
	// Each item carries the best-effort repo name (the repo is named "web").
	for _, it := range list {
		if it.RepoID != repoID {
			t.Errorf("repoId = %q, want %q", it.RepoID, repoID)
		}
		if it.RepoName != "web" {
			t.Errorf("repoName = %q, want web", it.RepoName)
		}
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

func TestDeleteRoutineOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	repoID := newRepoForRoutine(t, srv)

	// Create a (pending) run, then delete it → 204.
	resp := postJSON(t, srv.URL+"/repos/"+repoID+"/routines/approve-and-tag", map[string]any{"mrIid": 7})
	var run struct {
		ID string `json:"id"`
	}
	decodeBody(t, resp, &run)

	delResp := doDelete(t, srv.URL+"/routines/"+run.ID)
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", delResp.StatusCode)
	}

	// The run is gone: a follow-up GET is 404.
	getResp, err := http.Get(srv.URL + "/routines/" + run.ID)
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", getResp.StatusCode)
	}
}

func TestDeleteUnknownRoutineOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	resp := doDelete(t, srv.URL+"/routines/does-not-exist")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete unknown status = %d, want 404", resp.StatusCode)
	}
}

// newRoutineServerWithRunningRun wires a minimal server whose routine store holds
// a single run in the "running" status, returning the server and that run's id so
// the delete-conflict (409) wire contract can be exercised. Only the routines
// service is wired; every other dependency is nil (the delete path never touches
// them) and auth is disabled.
func newRoutineServerWithRunningRun(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "a", BaseURL: "u", TokenRef: "r", CreatedAt: time.Now().UTC()}
	if err := sqlite.NewAccountRepo(db).Create(ctx, acc); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	rp := repo.Repo{ID: id.New(), Name: "web", URL: "u", AccountID: acc.ID, CreatedAt: time.Now().UTC()}
	if err := sqlite.NewRepoStore(db).Create(ctx, rp); err != nil {
		t.Fatalf("seed repo: %v", err)
	}

	runStore := sqlite.NewRoutineRunStore(db)
	runID := id.New()
	run := routine.Run{
		ID:        runID,
		Kind:      routine.KindRelease,
		RepoID:    rp.ID,
		MRIID:     7,
		Status:    routine.RunRunning,
		Params:    json.RawMessage("{}"),
		State:     json.RawMessage("{}"),
		Steps:     []routine.Step{{Name: "verify", Status: routine.StepRunning}},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := runStore.Create(ctx, run); err != nil {
		t.Fatalf("seed running run: %v", err)
	}

	routinesSvc := routines.NewService(sqlite.NewRepoStore(db), nil, runStore, time.Minute, nil, log.New(io.Discard, "", 0))
	var set skills.Set
	srv := httptest.NewServer(NewServer(nil, nil, nil, nil, nil, routinesSvc, nil, nil, nil, set, nil, "", nil, false).Routes())
	t.Cleanup(srv.Close)
	return srv, runID
}

func TestDeleteRunningRoutineOverHTTP(t *testing.T) {
	srv, runID := newRoutineServerWithRunningRun(t)

	// A running run cannot be deleted mid-flight → 409 conflict.
	resp := doDelete(t, srv.URL+"/routines/"+runID)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("delete running status = %d, want 409", resp.StatusCode)
	}
}

func TestToRunExposesReleaseBranchInfo(t *testing.T) {
	run := routine.Run{
		ID:     "r1",
		Kind:   routine.KindRelease,
		RepoID: "repo1",
		Status: routine.RunPending,
		Params: json.RawMessage(`{"flow":"main","sourceBranch":"development","targetBranch":"main","bump":"minor"}`),
	}
	dto := toRun(run)
	if dto.Flow != "main" {
		t.Errorf("Flow = %q, want main", dto.Flow)
	}
	if dto.SourceBranch != "development" {
		t.Errorf("SourceBranch = %q, want development", dto.SourceBranch)
	}
	if dto.TargetBranch != "main" {
		t.Errorf("TargetBranch = %q, want main", dto.TargetBranch)
	}

	// A non-release run exposes none of the branch fields.
	other := toRun(routine.Run{ID: "r2", Kind: routine.KindApproveAndTag, Params: json.RawMessage(`{"bump":"patch"}`)})
	if other.Flow != "" || other.SourceBranch != "" || other.TargetBranch != "" {
		t.Errorf("non-release run leaked branch info: %+v", other)
	}
}

// TestCancelRoutineOverHTTP cancels a pending run and asserts the 200 response
// carries the updated run in the "cancelled" status, then that a re-cancel of the
// now-terminal run is a 409 conflict.
func TestCancelRoutineOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	repoID := newRepoForRoutine(t, srv)

	resp := postJSON(t, srv.URL+"/repos/"+repoID+"/routines/approve-and-tag", map[string]any{"mrIid": 7})
	var run struct {
		ID string `json:"id"`
	}
	decodeBody(t, resp, &run)

	cancelResp := postJSON(t, srv.URL+"/routines/"+run.ID+"/cancel", nil)
	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200", cancelResp.StatusCode)
	}
	var cancelled struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, cancelResp, &cancelled)
	if cancelled.ID != run.ID || cancelled.Status != "cancelled" {
		t.Fatalf("cancelled run = %+v, want id %q status cancelled", cancelled, run.ID)
	}

	// The run is now terminal: a second cancel is a 409 conflict.
	againResp := postJSON(t, srv.URL+"/routines/"+run.ID+"/cancel", nil)
	againResp.Body.Close()
	if againResp.StatusCode != http.StatusConflict {
		t.Fatalf("re-cancel status = %d, want 409", againResp.StatusCode)
	}

	// A cancelled run is deletable (only running is refused) → 204.
	delResp := doDelete(t, srv.URL+"/routines/"+run.ID)
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete cancelled status = %d, want 204", delResp.StatusCode)
	}
}

// TestCancelUnknownRoutineOverHTTP maps an unknown run to 404.
func TestCancelUnknownRoutineOverHTTP(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/routines/does-not-exist/cancel", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("cancel unknown status = %d, want 404", resp.StatusCode)
	}
}
