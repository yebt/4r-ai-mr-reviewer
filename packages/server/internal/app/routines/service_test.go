package routines

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	appRepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
)

func intPtr(v int) *int { return &v }

// fakeConfig configures the canned GitLab responses per endpoint. A zero status
// means 200; a non-2xx status makes that endpoint fail so the best-effort /
// fail-closed branches can be exercised.
type fakeConfig struct {
	tokenStatus   int
	tokenScopes   []string
	defaultBranch string
	projectAccess *int
	groupAccess   *int
	branchStatus  int
	branches      []map[string]any
	tagStatus     int
	tags          []map[string]any
}

// newFakeGitLab serves the four endpoints the preflight probes, routed by path.
func newFakeGitLab(t *testing.T, cfg fakeConfig) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal_access_tokens/self"):
			if cfg.tokenStatus != 0 && cfg.tokenStatus != http.StatusOK {
				w.WriteHeader(cfg.tokenStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"active": true, "scopes": cfg.tokenScopes})
		case strings.HasSuffix(r.URL.Path, "/protected_branches"):
			if cfg.branchStatus != 0 && cfg.branchStatus != http.StatusOK {
				w.WriteHeader(cfg.branchStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(cfg.branches)
		case strings.HasSuffix(r.URL.Path, "/protected_tags"):
			if cfg.tagStatus != 0 && cfg.tagStatus != http.StatusOK {
				w.WriteHeader(cfg.tagStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(cfg.tags)
		case strings.Contains(r.URL.Path, "/projects/"):
			perms := map[string]any{}
			if cfg.projectAccess != nil {
				perms["project_access"] = map[string]any{"access_level": *cfg.projectAccess}
			}
			if cfg.groupAccess != nil {
				perms["group_access"] = map[string]any{"access_level": *cfg.groupAccess}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"default_branch": cfg.defaultBranch,
				"permissions":    perms,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// setupRoutinesTest wires a routines.Service whose single repo's account points
// at baseURL, returning the service and the repo id.
func setupRoutinesTest(t *testing.T, baseURL string) (context.Context, *Service, string) {
	t.Helper()
	ctx := context.Background()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))

	acc, err := accountSvc.Add(ctx, "acc", baseURL, "token")
	if err != nil {
		t.Fatalf("account Add: %v", err)
	}
	rp, err := repoSvc.Add(ctx, appRepos.AddInput{
		Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID,
	})
	if err != nil {
		t.Fatalf("repo Add: %v", err)
	}

	svc := NewService(sqlite.NewRepoStore(db), accountSvc, sqlite.NewRoutineRunStore(db), log.New(io.Discard, "", 0))
	return ctx, svc, rp.ID
}

// findCheck returns the Check for a capability, failing the test if absent.
func findCheck(t *testing.T, pf routine.Preflight, cap routine.Capability) routine.Check {
	t.Helper()
	for _, c := range pf.Checks {
		if c.Capability == cap {
			return c
		}
	}
	t.Fatalf("no check for capability %q", cap)
	return routine.Check{}
}

func TestServicePreflight(t *testing.T) {
	tests := []struct {
		name  string
		cfg   fakeConfig
		check func(t *testing.T, pf routine.Preflight)
	}{
		{
			name: "happy: api scope, Maintainer, protected main → merge ok",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
				projectAccess: intPtr(40),
				branches: []map[string]any{
					{"name": "main", "merge_access_levels": []map[string]any{{"access_level": 40}}},
				},
				tags: []map[string]any{},
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if !pf.ScopesKnown {
					t.Errorf("ScopesKnown = false, want true")
				}
				if pf.AccessLevel != 40 {
					t.Errorf("AccessLevel = %d, want 40", pf.AccessLevel)
				}
				if c := findCheck(t, pf, routine.CapMergeMR); c.Status != routine.StatusOK {
					t.Errorf("merge_mr status = %q, want ok (detail %q)", c.Status, c.Detail)
				}
			},
		},
		{
			name: "group access outranks project access",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
				projectAccess: intPtr(30),
				groupAccess:   intPtr(40),
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if pf.AccessLevel != 40 {
					t.Errorf("AccessLevel = %d, want 40 (group outranks project)", pf.AccessLevel)
				}
				if pf.AccessLevelName != "Maintainer" {
					t.Errorf("AccessLevelName = %q, want Maintainer", pf.AccessLevelName)
				}
			},
		},
		{
			name: "no membership at any level → None",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if pf.AccessLevel != 0 {
					t.Errorf("AccessLevel = %d, want 0", pf.AccessLevel)
				}
				if pf.AccessLevelName != "None" {
					t.Errorf("AccessLevelName = %q, want None", pf.AccessLevelName)
				}
			},
		},
		{
			name: "protected-branches lookup fails → merge unknown (fail-closed)",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
				projectAccess: intPtr(40),
				branchStatus:  http.StatusForbidden,
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if c := findCheck(t, pf, routine.CapMergeMR); c.Status != routine.StatusUnknown {
					t.Errorf("merge_mr status = %q, want unknown when protection unreadable (detail %q)", c.Status, c.Detail)
				}
			},
		},
		{
			name: "token-self fails → scopes unknown",
			cfg: fakeConfig{
				tokenStatus:   http.StatusUnauthorized,
				defaultBranch: "main",
				projectAccess: intPtr(40),
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if pf.ScopesKnown {
					t.Errorf("ScopesKnown = true, want false when token-self fails")
				}
				if c := findCheck(t, pf, routine.CapComment); c.Status != routine.StatusUnknown {
					t.Errorf("comment status = %q, want unknown when scopes unverified", c.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeGitLab(t, tt.cfg)
			ctx, svc, repoID := setupRoutinesTest(t, srv.URL)
			pf, err := svc.Preflight(ctx, repoID)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			tt.check(t, pf)
		})
	}
}

func TestServicePreflightUnknownRepo(t *testing.T) {
	srv := newFakeGitLab(t, fakeConfig{tokenScopes: []string{"api"}, defaultBranch: "main"})
	ctx, svc, _ := setupRoutinesTest(t, srv.URL)
	if _, err := svc.Preflight(ctx, "does-not-exist"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("Preflight(unknown) error = %v, want repo.ErrNotFound", err)
	}
}

// --- approve_and_tag ---

// fakeGitLabState carries the mutable, thread-safe state a fake GitLab uses to
// serve the approve_and_tag write endpoints and record what a run did.
type fakeGitLabState struct {
	mu             sync.Mutex
	mrState        string
	targetBranch   string
	mergeCommitSHA string
	tags           []string
	awardCount     int
	noteCount      int
	createTagCount int
	lastCreateRef  string
	failAward      bool
	failCreateTag  bool
	createTagBody  string
}

// newFakeApproveGitLab serves the endpoints an approve_and_tag run exercises:
// the MR fetch, award_emoji, notes, and tag list/create.
func newFakeApproveGitLab(t *testing.T, st *fakeGitLabState) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		st.mu.Lock()
		defer st.mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/award_emoji") && r.Method == http.MethodPost:
			if st.failAward {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			st.awardCount++
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": st.awardCount})
		case strings.HasSuffix(r.URL.Path, "/notes") && r.Method == http.MethodPost:
			st.noteCount++
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": st.noteCount})
		case strings.HasSuffix(r.URL.Path, "/repository/tags") && r.Method == http.MethodPost:
			if st.failCreateTag {
				w.WriteHeader(http.StatusInternalServerError)
				if st.createTagBody != "" {
					_, _ = w.Write([]byte(st.createTagBody))
				}
				return
			}
			st.createTagCount++
			name := r.FormValue("tag_name")
			st.lastCreateRef = r.FormValue("ref")
			st.tags = append(st.tags, name)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name})
		case strings.HasSuffix(r.URL.Path, "/repository/tags") && r.Method == http.MethodGet:
			out := make([]map[string]any, 0, len(st.tags))
			for _, name := range st.tags {
				out = append(out, map[string]any{"name": name})
			}
			_ = json.NewEncoder(w).Encode(out)
		case strings.Contains(r.URL.Path, "/merge_requests/") && r.Method == http.MethodGet:
			state := st.mrState
			if state == "" {
				state = "opened"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"iid": 7, "state": state,
				"source_branch": "feature", "target_branch": st.targetBranch,
				"merge_commit_sha": st.mergeCommitSHA,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func stepByName(t *testing.T, run routine.Run, name string) routine.Step {
	t.Helper()
	for _, st := range run.Steps {
		if st.Name == name {
			return st
		}
	}
	t.Fatalf("no step %q in run", name)
	return routine.Step{}
}

func TestApproveAndTagFullRun(t *testing.T) {
	st := &fakeGitLabState{mrState: "merged", targetBranch: "main", mergeCommitSHA: "deadbeefsha", tags: []string{"1.2.3"}}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	if run.Status != routine.RunPending {
		t.Fatalf("created status = %q, want pending", run.Status)
	}

	// Drive the run synchronously (do not depend on the async worker's timing).
	if err := svc.execute(ctx, run); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got, err := svc.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != routine.RunDone {
		t.Fatalf("status = %q, want done (lastError %q)", got.Status, got.LastError)
	}
	if stepByName(t, got, "react").Status != routine.StepDone ||
		stepByName(t, got, "comment").Status != routine.StepDone ||
		stepByName(t, got, "tag").Status != routine.StepDone {
		t.Fatalf("not all steps done: %+v", got.Steps)
	}
	if st.awardCount != 2 { // default emojis: thumbsup, seedling
		t.Errorf("awardCount = %d, want 2", st.awardCount)
	}
	if st.noteCount != 1 {
		t.Errorf("noteCount = %d, want 1", st.noteCount)
	}
	if st.createTagCount != 1 || len(st.tags) != 2 || st.tags[1] != "1.2.4" {
		t.Fatalf("tag not created via NextTag: count=%d tags=%v", st.createTagCount, st.tags)
	}
	// The tag ref must be the merge commit SHA, not the target branch.
	if st.lastCreateRef != "deadbeefsha" {
		t.Errorf("tag ref = %q, want the merge commit SHA %q", st.lastCreateRef, "deadbeefsha")
	}
	// The accumulator persisted the computed tag.
	var state struct {
		NextTag string `json:"nextTag"`
	}
	if err := json.Unmarshal(got.State, &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if state.NextTag != "1.2.4" {
		t.Errorf("state.nextTag = %q, want 1.2.4", state.NextTag)
	}
}

func TestApproveAndTagBlocksThenResumes(t *testing.T) {
	st := &fakeGitLabState{mrState: "merged", targetBranch: "main", mergeCommitSHA: "deadbeefsha", tags: []string{"1.2.3"}, failCreateTag: true}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}

	// First pass: the tag step fails, so the run blocks with react+comment done.
	if err := svc.execute(ctx, run); err != nil {
		t.Fatalf("execute: %v", err)
	}
	blocked, _ := svc.Get(ctx, run.ID)
	if blocked.Status != routine.RunBlocked {
		t.Fatalf("status = %q, want blocked", blocked.Status)
	}
	if stepByName(t, blocked, "react").Status != routine.StepDone ||
		stepByName(t, blocked, "comment").Status != routine.StepDone {
		t.Fatalf("earlier steps should be done: %+v", blocked.Steps)
	}
	if stepByName(t, blocked, "tag").Status != routine.StepFailed {
		t.Fatalf("tag step = %q, want failed", stepByName(t, blocked, "tag").Status)
	}
	if blocked.LastError == "" {
		t.Errorf("blocked run should record last_error")
	}
	if st.noteCount != 1 {
		t.Fatalf("noteCount after first pass = %d, want 1", st.noteCount)
	}

	// Fix the cause and resume: the checkpointed steps must NOT re-run.
	st.mu.Lock()
	st.failCreateTag = false
	st.mu.Unlock()

	resumed, err := svc.Resume(ctx, run.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.Status != routine.RunPending {
		t.Fatalf("resumed status = %q, want pending", resumed.Status)
	}
	if err := svc.execute(ctx, resumed); err != nil {
		t.Fatalf("execute (resume): %v", err)
	}

	done, _ := svc.Get(ctx, run.ID)
	if done.Status != routine.RunDone {
		t.Fatalf("status = %q, want done after resume (lastError %q)", done.Status, done.LastError)
	}
	// The comment was posted exactly once across both passes — proof the
	// checkpoint skipped the already-done step.
	if st.noteCount != 1 {
		t.Errorf("noteCount = %d, want 1 (comment must not repeat on resume)", st.noteCount)
	}
	if st.createTagCount != 1 || len(st.tags) != 2 || st.tags[1] != "1.2.4" {
		t.Fatalf("tag not created on resume: count=%d tags=%v", st.createTagCount, st.tags)
	}
}

func TestResumeRejectsNonBlockedRun(t *testing.T) {
	st := &fakeGitLabState{targetBranch: "main"}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	if _, err := svc.Resume(ctx, run.ID); !errors.Is(err, routine.ErrNotResumable) {
		t.Fatalf("Resume(pending) = %v, want ErrNotResumable", err)
	}
}

func TestCreateApproveAndTagRejectsBadBump(t *testing.T) {
	st := &fakeGitLabState{targetBranch: "main"}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)
	if _, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7, Bump: "nope"}); err == nil {
		t.Fatal("expected a validation error for an invalid bump")
	}
}

func TestCreateApproveAndTagUnknownRepo(t *testing.T) {
	st := &fakeGitLabState{targetBranch: "main"}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, _ := setupRoutinesTest(t, srv.URL)
	if _, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: "nope", MRIID: 7}); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("CreateApproveAndTag(unknown repo) = %v, want repo.ErrNotFound", err)
	}
}

// Fix B: an unmerged MR blocks the run at the tag step, after react+comment ran.
func TestApproveAndTagBlocksWhenMRNotMerged(t *testing.T) {
	st := &fakeGitLabState{mrState: "opened", targetBranch: "main", tags: []string{"1.2.3"}}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	if err := svc.execute(ctx, run); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got, _ := svc.Get(ctx, run.ID)
	if got.Status != routine.RunBlocked {
		t.Fatalf("status = %q, want blocked (lastError %q)", got.Status, got.LastError)
	}
	if stepByName(t, got, "react").Status != routine.StepDone ||
		stepByName(t, got, "comment").Status != routine.StepDone {
		t.Fatalf("react and comment should be done before the tag block: %+v", got.Steps)
	}
	if stepByName(t, got, "tag").Status != routine.StepFailed {
		t.Fatalf("tag step = %q, want failed", stepByName(t, got, "tag").Status)
	}
	if !strings.Contains(got.LastError, "not merged") {
		t.Errorf("lastError = %q, want it to mention 'not merged'", got.LastError)
	}
	if st.createTagCount != 0 {
		t.Errorf("createTagCount = %d, want 0 (must not tag an unmerged MR)", st.createTagCount)
	}
}

// Fix A: a crash after CreateTag but before the step checkpoint must not create a
// second, higher tag on resume. We drive a full pass (tag created once), then
// simulate the lost step-status checkpoint by flipping the tag step back to
// running with its NextTag decision still persisted, and re-run.
func TestApproveAndTagResumeAfterTagCreatedIsIdempotent(t *testing.T) {
	st := &fakeGitLabState{mrState: "merged", targetBranch: "main", mergeCommitSHA: "sha1", tags: []string{"1.2.3"}}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	if err := svc.execute(ctx, run); err != nil {
		t.Fatalf("execute (first pass): %v", err)
	}
	if st.createTagCount != 1 {
		t.Fatalf("createTagCount after first pass = %d, want 1", st.createTagCount)
	}

	// Simulate the crash window: the tag exists (1.2.4) and NextTag is persisted,
	// but the tag step never recorded "done".
	crashed, _ := svc.Get(ctx, run.ID)
	for i := range crashed.Steps {
		if crashed.Steps[i].Name == stepTag {
			crashed.Steps[i].Status = routine.StepRunning
		}
	}
	crashed.Status = routine.RunRunning
	if err := svc.runs.Save(ctx, crashed); err != nil {
		t.Fatalf("Save (simulate crash): %v", err)
	}

	if err := svc.execute(ctx, crashed); err != nil {
		t.Fatalf("execute (resume): %v", err)
	}
	if st.createTagCount != 1 {
		t.Fatalf("createTagCount after resume = %d, want 1 (no second, higher tag)", st.createTagCount)
	}
	if len(st.tags) != 2 || st.tags[1] != "1.2.4" {
		t.Fatalf("tags = %v, want a single 1.2.4 bump", st.tags)
	}
	done, _ := svc.Get(ctx, run.ID)
	if done.Status != routine.RunDone {
		t.Fatalf("status = %q, want done", done.Status)
	}
}

// Fix (ledger): a failure on the FIRST step leaves later steps Pending.
func TestApproveAndTagBlockOnFirstStepLeavesLaterStepsPending(t *testing.T) {
	st := &fakeGitLabState{mrState: "merged", targetBranch: "main", mergeCommitSHA: "sha1", failAward: true}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	if err := svc.execute(ctx, run); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got, _ := svc.Get(ctx, run.ID)
	if got.Status != routine.RunBlocked {
		t.Fatalf("status = %q, want blocked", got.Status)
	}
	if stepByName(t, got, "react").Status != routine.StepFailed {
		t.Fatalf("react step = %q, want failed", stepByName(t, got, "react").Status)
	}
	if stepByName(t, got, "comment").Status != routine.StepPending ||
		stepByName(t, got, "tag").Status != routine.StepPending {
		t.Fatalf("comment and tag should stay pending: %+v", got.Steps)
	}
	if st.noteCount != 0 || st.createTagCount != 0 {
		t.Errorf("no later actions should run: noteCount=%d createTagCount=%d", st.noteCount, st.createTagCount)
	}
}

// Fix D: a computed tag matches an existing one across a "v" prefix mismatch, so
// no duplicate tag is created.
func TestApproveAndTagRecognizesExistingTagAcrossVPrefix(t *testing.T) {
	st := &fakeGitLabState{mrState: "merged", targetBranch: "main", mergeCommitSHA: "sha1", tags: []string{"v1.2.4"}}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	// Seed the durable decision as the non-v form to force the prefix mismatch.
	seeded, _ := svc.Get(ctx, run.ID)
	seeded.State = json.RawMessage(`{"nextTag":"1.2.4"}`)
	for i := range seeded.Steps {
		switch seeded.Steps[i].Name {
		case stepReact, stepComment:
			seeded.Steps[i].Status = routine.StepDone
		case stepTag:
			seeded.Steps[i].Status = routine.StepRunning
		}
	}
	seeded.Status = routine.RunRunning
	if err := svc.runs.Save(ctx, seeded); err != nil {
		t.Fatalf("Save (seed): %v", err)
	}

	if err := svc.execute(ctx, seeded); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if st.createTagCount != 0 {
		t.Errorf("createTagCount = %d, want 0 (v1.2.4 already exists for computed 1.2.4)", st.createTagCount)
	}
	done, _ := svc.Get(ctx, run.ID)
	if done.Status != routine.RunDone {
		t.Fatalf("status = %q, want done", done.Status)
	}
}

func TestTagExistsNormalizesVPrefix(t *testing.T) {
	if !tagExists([]string{"v1.2.4"}, "1.2.4") {
		t.Error("tagExists(v-prefixed existing, plain computed) = false, want true")
	}
	if !tagExists([]string{"1.2.4"}, "v1.2.4") {
		t.Error("tagExists(plain existing, v-prefixed computed) = false, want true")
	}
	if tagExists([]string{"v1.2.3", "v1.2.4"}, "v1.2.5") {
		t.Error("tagExists reported a match for a genuinely new tag")
	}
}

// Fix E: a non-positive MR IID is rejected at creation (HTTP 400, not a NULL row).
func TestCreateApproveAndTagRejectsNonPositiveMRIID(t *testing.T) {
	st := &fakeGitLabState{targetBranch: "main"}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)
	for _, iid := range []int{0, -3} {
		if _, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: iid}); err == nil {
			t.Fatalf("MRIID=%d: expected a validation error", iid)
		}
	}
}

// Fix G: a second active run for the same MR is rejected with ErrDuplicateRun.
func TestCreateApproveAndTagRejectsDuplicateActiveRun(t *testing.T) {
	st := &fakeGitLabState{targetBranch: "main"}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	if _, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7}); err != nil {
		t.Fatalf("first CreateApproveAndTag: %v", err)
	}
	if _, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7}); !errors.Is(err, routine.ErrDuplicateRun) {
		t.Fatalf("duplicate CreateApproveAndTag = %v, want ErrDuplicateRun", err)
	}
	// A different MR is unaffected.
	if _, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 8}); err != nil {
		t.Fatalf("CreateApproveAndTag for a different MR: %v", err)
	}
}

// Fix F: a GitLab APIError does not leak its raw body into the persisted
// LastError; only the sanitized status form is stored/served.
func TestApproveAndTagSanitizesUpstreamError(t *testing.T) {
	st := &fakeGitLabState{
		mrState: "merged", targetBranch: "main", mergeCommitSHA: "sha1", tags: []string{"1.2.3"},
		failCreateTag: true,
		createTagBody: "internal detail line one\nSENSITIVE_SECRET_TOKEN_xyz",
	}
	srv := newFakeApproveGitLab(t, st)
	ctx, svc, repoID := setupRoutinesTest(t, srv.URL)

	run, err := svc.CreateApproveAndTag(ctx, ApproveAndTagInput{RepoID: repoID, MRIID: 7})
	if err != nil {
		t.Fatalf("CreateApproveAndTag: %v", err)
	}
	if err := svc.execute(ctx, run); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got, _ := svc.Get(ctx, run.ID)
	if got.Status != routine.RunBlocked {
		t.Fatalf("status = %q, want blocked", got.Status)
	}
	if !strings.Contains(got.LastError, "gitlab returned status 500") {
		t.Errorf("lastError = %q, want the sanitized status form", got.LastError)
	}
	if strings.Contains(got.LastError, "SENSITIVE_SECRET_TOKEN_xyz") {
		t.Errorf("lastError leaked the raw upstream body: %q", got.LastError)
	}
	if d := stepByName(t, got, "tag").Detail; strings.Contains(d, "SENSITIVE_SECRET_TOKEN_xyz") {
		t.Errorf("tag step detail leaked the raw upstream body: %q", d)
	}
}
