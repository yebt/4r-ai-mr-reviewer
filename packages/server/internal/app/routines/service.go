// Package routines coordinates merge-request routines. Its first responsibility
// is the preflight: reporting which routine actions a repo's token and access
// level allow, before any routine tries to run them.
package routines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// pollInterval bounds how long the single worker sleeps between sweeps when no
// wake-up arrives; it caps crash-recovery latency.
const pollInterval = 2 * time.Second

// Fixed step ledger for the approve_and_tag routine, executed in this order.
const (
	stepReact   = "react"
	stepComment = "comment"
	stepTag     = "tag"
)

// Service computes routine preflights and drives routine runs for tracked repos.
type Service struct {
	repos    repo.Repository
	accounts *accounts.Service
	runs     routine.RunStore
	logger   *log.Logger
	// signal wakes the worker when a run is created or resumed, so it need not
	// wait for the poll ticker.
	signal chan struct{}
}

// NewService wires the routines service. A nil logger defaults to the standard
// logger.
func NewService(repos repo.Repository, accounts *accounts.Service, runs routine.RunStore, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		repos:    repos,
		accounts: accounts,
		runs:     runs,
		logger:   logger,
		signal:   make(chan struct{}, 1),
	}
}

// approveAndTagParams is the immutable input persisted in Run.Params.
type approveAndTagParams struct {
	Emojis  []string `json:"emojis"`
	Comment string   `json:"comment"`
	Bump    string   `json:"bump"`
}

// approveAndTagState is the mutable accumulator persisted in Run.State.
type approveAndTagState struct {
	NextTag string `json:"nextTag,omitempty"`
}

// Preflight probes a repo's GitLab project and reports which routine actions the
// account's token and access level permit. It returns repo.ErrNotFound when the
// repo is unknown.
//
// TokenSelf, ProtectedBranches, and ProtectedTags are best-effort: a token type
// that cannot introspect scopes, or a project without protection rules, must not
// fail the whole preflight — only Project is required (it yields the access
// level and default branch the checks hinge on).
func (s *Service) Preflight(ctx context.Context, repoID string) (routine.Preflight, error) {
	gl, projectID, err := s.gitlabFor(ctx, repoID)
	if err != nil {
		return routine.Preflight{}, err
	}

	in := routine.PreflightInput{}

	// Token scopes are unknown for OAuth/job/deploy tokens; tolerate the error.
	if info, err := gl.TokenSelf(ctx); err != nil {
		in.ScopesKnown = false
		in.Scopes = nil
	} else {
		in.ScopesKnown = true
		in.Scopes = info.Scopes
	}

	// Project is required: it carries the effective access level and default branch.
	project, err := gl.Project(ctx, projectID)
	if err != nil {
		return routine.Preflight{}, err
	}
	in.AccessLevel = effectiveAccessLevel(project.Permissions)
	in.DefaultBranch = project.DefaultBranch

	// Branch/tag protection is best-effort, but fail-closed: on failure the
	// *Known flag stays false so the preflight reports "unknown" rather than
	// silently assuming the branch/tag is unprotected.
	if branches, err := gl.ProtectedBranches(ctx, projectID); err != nil {
		log.Printf("routines: preflight: protected branches for repo %s: %v", repoID, err)
	} else {
		in.ProtectedBranches = protectedBranchLevels(branches)
		in.BranchProtectionKnown = true
	}
	if tags, err := gl.ProtectedTags(ctx, projectID); err != nil {
		log.Printf("routines: preflight: protected tags for repo %s: %v", repoID, err)
	} else {
		in.ProtectedTagsMinLevel = protectedTagsMinLevel(tags)
		in.TagProtectionKnown = true
	}

	return routine.Evaluate(in), nil
}

// gitlabFor builds a GitLab client for a repo's account and returns the client
// and encoded project path. It returns repo.ErrNotFound when the repo is
// unknown. This mirrors reviews.gitlabFor, which is unexported in the reviews
// package, so the ~5 lines are duplicated deliberately.
func (s *Service) gitlabFor(ctx context.Context, repoID string) (*gitlab.Client, string, error) {
	rp, err := s.repos.Get(ctx, repoID)
	if err != nil {
		return nil, "", err
	}
	acc, err := s.accounts.Get(ctx, rp.AccountID)
	if err != nil {
		return nil, "", err
	}
	token, err := s.accounts.Token(ctx, rp.AccountID)
	if err != nil {
		return nil, "", err
	}
	projectID, err := gitlab.ProjectPath(rp.URL)
	if err != nil {
		return nil, "", err
	}
	return gitlab.NewClient(acc.BaseURL, token), projectID, nil
}

// effectiveAccessLevel is the highest of the caller's direct project access and
// inherited group access (either may be nil).
func effectiveAccessLevel(p gitlab.ProjectPermissions) int {
	level := 0
	if p.ProjectAccess != nil && p.ProjectAccess.AccessLevel > level {
		level = p.ProjectAccess.AccessLevel
	}
	if p.GroupAccess != nil && p.GroupAccess.AccessLevel > level {
		level = p.GroupAccess.AccessLevel
	}
	return level
}

// protectedBranchLevels maps each protected branch to the highest access level
// required to merge into it.
func protectedBranchLevels(branches []gitlab.ProtectedBranch) map[string]int {
	out := make(map[string]int, len(branches))
	for _, b := range branches {
		highest := 0
		for _, rule := range b.MergeAccessLevels {
			if rule.AccessLevel > highest {
				highest = rule.AccessLevel
			}
		}
		out[b.Name] = highest
	}
	return out
}

// protectedTagsMinLevel is the highest create access level across all protected
// tag rules (0 when there are no protected tags).
func protectedTagsMinLevel(tags []gitlab.ProtectedTag) int {
	highest := 0
	for _, t := range tags {
		for _, rule := range t.CreateAccessLevels {
			if rule.AccessLevel > highest {
				highest = rule.AccessLevel
			}
		}
	}
	return highest
}

// --- approve_and_tag routine ---

// ApproveAndTagInput is the request to create an approve_and_tag run. Emojis,
// Comment and Bump are optional and fall back to defaults.
type ApproveAndTagInput struct {
	RepoID  string
	MRIID   int
	Emojis  []string
	Comment string
	Bump    string
}

// validBumps is the set of accepted semver bump levels.
var validBumps = map[string]bool{"major": true, "minor": true, "patch": true}

// CreateApproveAndTag validates the repo, applies defaults, records a pending
// run with the fixed react/comment/tag ledger, and wakes the worker. It returns
// repo.ErrNotFound for an unknown repo and a validation error for a bad bump.
func (s *Service) CreateApproveAndTag(ctx context.Context, in ApproveAndTagInput) (routine.Run, error) {
	if _, err := s.repos.Get(ctx, in.RepoID); err != nil {
		return routine.Run{}, err
	}

	// A non-positive IID would persist as a NULL row that only surfaces later as a
	// blocked run; reject it up front so the HTTP layer returns 400.
	if in.MRIID <= 0 {
		return routine.Run{}, fmt.Errorf("routines: invalid merge request iid %d (must be positive)", in.MRIID)
	}

	emojis := in.Emojis
	if len(emojis) == 0 {
		emojis = []string{"thumbsup", "seedling"}
	}
	comment := in.Comment
	if comment == "" {
		comment = "LGFM"
	}
	bump := in.Bump
	if bump == "" {
		bump = "patch"
	}
	if !validBumps[bump] {
		return routine.Run{}, fmt.Errorf("routines: invalid bump %q (want major, minor or patch)", bump)
	}

	// Reject a duplicate: an active run for the same MR would re-post the same
	// reactions/comment and re-tag, spamming the merge request.
	runs, err := s.runs.ListByRepo(ctx, in.RepoID)
	if err != nil {
		return routine.Run{}, err
	}
	for _, r := range runs {
		if r.Kind == routine.KindApproveAndTag && r.MRIID == in.MRIID && isActiveRun(r.Status) {
			return routine.Run{}, routine.ErrDuplicateRun
		}
	}

	params, err := json.Marshal(approveAndTagParams{Emojis: emojis, Comment: comment, Bump: bump})
	if err != nil {
		return routine.Run{}, err
	}

	now := time.Now().UTC()
	run := routine.Run{
		ID:     id.New(),
		Kind:   routine.KindApproveAndTag,
		RepoID: in.RepoID,
		MRIID:  in.MRIID,
		Status: routine.RunPending,
		Params: params,
		State:  json.RawMessage("{}"),
		Steps: []routine.Step{
			{Name: stepReact, Status: routine.StepPending},
			{Name: stepComment, Status: routine.StepPending},
			{Name: stepTag, Status: routine.StepPending},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.runs.Create(ctx, run); err != nil {
		return routine.Run{}, err
	}
	s.wake()
	return run, nil
}

// Get returns a run by id.
func (s *Service) Get(ctx context.Context, runID string) (routine.Run, error) {
	return s.runs.Get(ctx, runID)
}

// ListByRepo returns a repo's runs, newest first.
func (s *Service) ListByRepo(ctx context.Context, repoID string) ([]routine.Run, error) {
	return s.runs.ListByRepo(ctx, repoID)
}

// Resume re-queues a blocked run and wakes the worker. A run that is not blocked
// returns routine.ErrNotResumable. The checkpointed steps mean the resumed run
// skips the actions it already completed.
func (s *Service) Resume(ctx context.Context, runID string) (routine.Run, error) {
	run, err := s.runs.Get(ctx, runID)
	if err != nil {
		return routine.Run{}, err
	}
	if run.Status != routine.RunBlocked {
		return routine.Run{}, routine.ErrNotResumable
	}
	run.Status = routine.RunPending
	run.LastError = ""
	if err := s.runs.Save(ctx, run); err != nil {
		return routine.Run{}, err
	}
	s.wake()
	return run, nil
}

// Start recovers interrupted runs and drives the worker loop until ctx is done.
// It blocks; run it in its own goroutine. A SINGLE worker is intentional: it
// serializes runs so two routines never race the same repo's tags.
func (s *Service) Start(ctx context.Context) {
	if err := s.runs.RequeueRunning(ctx); err != nil {
		s.logger.Printf("routines: requeue running failed: %v", err)
	}
	s.worker(ctx)
}

// worker claims and executes one run at a time until ctx is done.
func (s *Service) worker(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		run, ok, err := s.runs.ClaimPending(ctx)
		if err != nil {
			s.logger.Printf("routines: claim failed: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-s.signal:
			case <-ticker.C:
			}
			continue
		}
		s.safeExecute(ctx, run)
	}
}

// wake signals the worker without blocking if a signal is already pending.
func (s *Service) wake() {
	select {
	case s.signal <- struct{}{}:
	default:
	}
}

// safeExecute runs execute, converting a panic into a logged error so one bad
// run cannot crash the process (there is no external supervisor to restart it).
func (s *Service) safeExecute(ctx context.Context, run routine.Run) {
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Printf("routines: run %s panicked: %v\n%s", run.ID, rec, debug.Stack())
		}
	}()
	if err := s.execute(ctx, run); err != nil {
		s.logger.Printf("routines: run %s execute failed: %v", run.ID, err)
	}
}

// execute drives a run's steps to completion, checkpointing after each. A failed
// step blocks the run (an expected pause, returned as nil so the worker keeps
// going); a genuine persistence failure is returned as an error.
func (s *Service) execute(ctx context.Context, run routine.Run) error {
	// A client-build failure is not the run's fault — pause it so a fixed token
	// or account lets a resume retry.
	gl, projectID, err := s.gitlabFor(ctx, run.RepoID)
	if err != nil {
		return s.block(ctx, run, fmt.Errorf("gitlab client: %w", err))
	}

	var params approveAndTagParams
	if err := json.Unmarshal(run.Params, &params); err != nil {
		return s.block(ctx, run, fmt.Errorf("decode params: %w", err))
	}

	// Fetch the MR fresh on every pass. This is intentional: the tag step requires
	// the MR to be merged, so a resume after the user merges must observe the new
	// "merged" state (and the merge commit SHA) to proceed.
	mr, err := gl.GetMergeRequest(ctx, projectID, run.MRIID)
	if err != nil {
		return s.block(ctx, run, fmt.Errorf("get merge request: %w", err))
	}

	var state approveAndTagState
	if len(run.State) > 0 {
		_ = json.Unmarshal(run.State, &state)
	}

	// checkpoint persists a mid-step decision (the computed tag) while the step is
	// still running, so a crash between the decision and its side effect cannot
	// recompute a different value on resume.
	checkpoint := func(st *approveAndTagState) error {
		run.State = marshalState(*st)
		return s.runs.Save(ctx, run)
	}

	run.Status = routine.RunRunning
	for i := range run.Steps {
		step := &run.Steps[i]
		if step.Status == routine.StepDone || step.Status == routine.StepSkipped {
			continue
		}

		step.Status = routine.StepRunning
		step.UpdatedAt = time.Now().UTC()
		run.State = marshalState(state)
		if err := s.runs.Save(ctx, run); err != nil {
			return fmt.Errorf("routines: save run %s: %w", run.ID, err)
		}

		detail, herr := s.runStep(ctx, gl, projectID, run.MRIID, step.Name, params, mr, &state, checkpoint)
		step.UpdatedAt = time.Now().UTC()
		run.State = marshalState(state)
		if herr != nil {
			safe := s.clientSafeError(herr)
			step.Status = routine.StepFailed
			step.Detail = safe
			run.Status = routine.RunBlocked
			run.LastError = safe
			if err := s.runs.Save(ctx, run); err != nil {
				return fmt.Errorf("routines: save run %s: %w", run.ID, err)
			}
			s.logger.Printf("routines: run %s blocked at step %q: %v", run.ID, step.Name, herr)
			return nil
		}

		step.Status = routine.StepDone
		step.Detail = detail
		if err := s.runs.Save(ctx, run); err != nil {
			return fmt.Errorf("routines: save run %s: %w", run.ID, err)
		}
	}

	run.Status = routine.RunDone
	run.LastError = ""
	run.State = marshalState(state)
	if err := s.runs.Save(ctx, run); err != nil {
		return fmt.Errorf("routines: save run %s: %w", run.ID, err)
	}
	return nil
}

// runStep executes one approve_and_tag step (check-then-act) and returns a short
// human-readable detail on success. It may mutate state (the accumulator).
func (s *Service) runStep(ctx context.Context, gl *gitlab.Client, projectID string, mrIID int, name string, params approveAndTagParams, mr gitlab.MergeRequest, state *approveAndTagState, checkpoint func(*approveAndTagState) error) (string, error) {
	switch name {
	case stepReact:
		for _, emoji := range params.Emojis {
			// AwardEmoji is idempotent: a duplicate award is swallowed, so a
			// re-run does not fail on a reaction set on a previous pass.
			if err := gl.AwardEmoji(ctx, projectID, mrIID, emoji); err != nil {
				return "", fmt.Errorf("award emoji %q: %w", emoji, err)
			}
		}
		return "reacted", nil

	case stepComment:
		// Idempotency here relies on step-checkpointing: a crash after posting but
		// before the checkpoint could double-post. Acceptable for a short "LGFM".
		if err := gl.CreateNote(ctx, projectID, mrIID, params.Comment); err != nil {
			return "", fmt.Errorf("create note: %w", err)
		}
		return "commented", nil

	case stepTag:
		// ASSUMPTIONS (kept here so they are easy to change later):
		//   - the MR must be MERGED before we tag; an unmerged MR blocks the run
		//     (an expected pause: the user merges, then resumes);
		//   - the tag REF is the merge commit SHA (merge_commit_sha, then
		//     squash_commit_sha, then the MR head sha) — never a moving branch head;
		//   - the bump defaults to patch (set at creation);
		//   - a "-development" prerelease suffix is applied when the target branch
		//     is "development".
		// The next tag is computed Go-side via routine.NextTag rather than trusting
		// the ListTags API ordering.
		const devBranch = "development"

		if mr.State != "merged" {
			return "", fmt.Errorf("merge request !%d is not merged yet (state %q); merge it and resume", mrIID, mr.State)
		}

		// The tag name is a durable decision: compute it once, persist it BEFORE
		// creating the tag, and reuse it on any resume. Recomputing after a crash
		// that already created the tag would pick a higher (wrongly bumped) value.
		if state.NextTag == "" {
			tags, _ := gl.ListTags(ctx, projectID)
			existing := make([]string, 0, len(tags))
			for _, t := range tags {
				existing = append(existing, t.Name)
			}
			prerelease := ""
			if mr.TargetBranch == devBranch {
				prerelease = devBranch
			}
			next, err := routine.NextTag(existing, params.Bump, prerelease)
			if err != nil {
				return "", fmt.Errorf("compute next tag: %w", err)
			}
			state.NextTag = next
			if err := checkpoint(state); err != nil {
				return "", fmt.Errorf("checkpoint next tag: %w", err)
			}
		}

		// Resolve the tag ref to the merge commit, preferring the merge commit,
		// then the squash commit, then the MR head SHA.
		ref := mr.MergeCommitSHA
		if ref == "" {
			ref = mr.SquashCommitSHA
		}
		if ref == "" {
			ref = mr.SHA
		}
		if ref == "" {
			return "", fmt.Errorf("merge request !%d is merged but exposes no commit SHA to tag", mrIID)
		}

		// Check-then-act against the durable decision: on resume the tag created by
		// a prior attempt is recognized (across a "v" prefix mismatch) and no second
		// tag is created.
		tags, _ := gl.ListTags(ctx, projectID)
		existing := make([]string, 0, len(tags))
		for _, t := range tags {
			existing = append(existing, t.Name)
		}
		if tagExists(existing, state.NextTag) {
			return fmt.Sprintf("tag %s already exists", state.NextTag), nil
		}
		if err := gl.CreateTag(ctx, projectID, state.NextTag, ref, ""); err != nil {
			return "", fmt.Errorf("create tag %s: %w", state.NextTag, err)
		}
		return fmt.Sprintf("created tag %s", state.NextTag), nil

	default:
		return "", fmt.Errorf("unknown step %q", name)
	}
}

// block records a run as blocked with a reason and persists it. A blocked run is
// an expected pause, so the worker treats the returned (nil-or-persist) error
// accordingly.
func (s *Service) block(ctx context.Context, run routine.Run, cause error) error {
	run.Status = routine.RunBlocked
	run.LastError = s.clientSafeError(cause)
	s.logger.Printf("routines: run %s blocked: %v", run.ID, cause)
	if err := s.runs.Save(ctx, run); err != nil {
		return fmt.Errorf("routines: save run %s: %w", run.ID, err)
	}
	return nil
}

// marshalState encodes the accumulator, falling back to an empty object so
// Run.State is always valid JSON.
func marshalState(state approveAndTagState) json.RawMessage {
	b, err := json.Marshal(state)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

// isActiveRun reports whether a run is still in flight (not done) and therefore
// blocks creating another run for the same merge request.
func isActiveRun(status routine.RunStatus) bool {
	switch status {
	case routine.RunPending, routine.RunRunning, routine.RunBlocked:
		return true
	default:
		return false
	}
}

// tagExists reports whether name already appears in existing, treating an
// optional single leading "v" as insignificant so a computed "1.2.4" matches an
// existing "v1.2.4" (and vice versa) and no duplicate tag is created.
func tagExists(existing []string, name string) bool {
	want := strings.TrimPrefix(name, "v")
	for _, x := range existing {
		if strings.TrimPrefix(x, "v") == want {
			return true
		}
	}
	return false
}

// clientSafeError renders err for values that are persisted in a run's LastError
// / step Detail and served over the unauthenticated run DTO. A wrapped
// *gitlab.APIError may carry sensitive upstream body detail, so it is reduced to
// its status (plus at most a short first line) and the full error is logged
// server-side; our own messages are safe and returned verbatim.
func (s *Service) clientSafeError(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *gitlab.APIError
	if errors.As(err, &apiErr) {
		s.logger.Printf("routines: upstream gitlab error (status %d): %v", apiErr.Status, err)
		msg := fmt.Sprintf("gitlab returned status %d", apiErr.Status)
		if line := firstLine(apiErr.Body); line != "" {
			if len(line) > 160 {
				line = line[:160]
			}
			msg += ": " + line
		}
		return msg
	}
	return err.Error()
}

// firstLine returns the first non-empty trimmed line of s.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
