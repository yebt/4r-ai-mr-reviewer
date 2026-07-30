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

// Fixed step ledger for the release routine, executed in this order. It reuses
// stepReact and stepTag (same names) and adds the release-specific steps. Both
// flows pause on the shared confirm gate before merging: the dev flow uses
// verify→...→confirm→merge; the main flow uses compute_tag→create_mr→
// wait_pipeline→approve→react→confirm→merge→tag→notify.
const (
	stepVerify       = "verify"
	stepApprove      = "approve"
	stepComputeTag   = "compute_tag"
	stepConfirm      = "confirm"
	stepMerge        = "merge"
	stepNotify       = "notify"
	stepCreateMR     = "create_mr"
	stepWaitPipeline = "wait_pipeline"
)

// Release flows. The dev flow drives an existing MR into "development" and tags a
// "-dev" prerelease; the main flow CREATES an MR from development into main and
// tags a pure release (no suffix, no confirmation gate).
const (
	flowDevelopment = "development"
	flowMain        = "main"
)

// devBranch is the source/target branch of the dev flow (an MR targeting it). Its
// tags carry a "-dev" prerelease suffix.
const devBranch = "development"

// mainBranch is the default target branch of the main flow.
const mainBranch = "main"

// mainReleaseMarker prefixes every MR title the main flow creates. runCreateMRStep
// only ever reuses an open source→target MR whose title carries this marker (one
// this routine opened on a prior attempt); an unmarked human-opened MR for the
// same branch pair blocks the run rather than being adopted and merged.
const mainReleaseMarker = "Main Release:"

// defaultMergePollInterval is how long the merge step waits between polls of the
// MR state while waiting for it to be merged. It is a field on Service so tests
// can shorten it.
const defaultMergePollInterval = 10 * time.Second

// errAwaitConfirmation is a control sentinel (not a failure): the confirm step
// returns it to pause a run on the interactive gate. execute recognizes it and
// flips the run to awaiting_confirmation WITHOUT blocking or failing the step, so
// a later Confirm re-enters the same step.
var errAwaitConfirmation = errors.New("routines: awaiting merge confirmation")

// errNothingToRelease is a control sentinel (not a failure): runComputeTagMain
// returns it when there are no releasable commits since the last release tag.
// executeRelease recognizes it, marks the compute step done with an explanatory
// detail, and completes the run WITHOUT opening an MR or running later steps.
var errNothingToRelease = errors.New("routines: no releasable commits")

// ReleaseNotifier delivers a best-effort, human-readable summary when a release
// routine finishes. It is optional: a nil notifier disables notification (the
// notify step just logs). Real routing (e.g. Telegram) is wired in a later slice.
type ReleaseNotifier interface {
	Notify(ctx context.Context, text string) error
}

// Service computes routine preflights and drives routine runs for tracked repos.
type Service struct {
	repos    repo.Repository
	accounts *accounts.Service
	runs     routine.RunStore
	logger   *log.Logger
	// signal wakes the worker when a run is created or resumed, so it need not
	// wait for the poll ticker.
	signal chan struct{}
	// mergeWaitTimeout bounds how long the release merge step waits for the MR to
	// be merged before it blocks (and can be resumed to keep waiting).
	mergeWaitTimeout time.Duration
	// mergePollInterval is the sleep between merge-state polls. It defaults to
	// defaultMergePollInterval and is overridable (tests shorten it).
	mergePollInterval time.Duration
	// notifier delivers the release summary; nil disables notification.
	notifier ReleaseNotifier
}

// NewService wires the routines service. A nil logger defaults to the standard
// logger; a non-positive mergeWaitTimeout defaults to 10 minutes; a nil notifier
// disables release notification.
func NewService(repos repo.Repository, accounts *accounts.Service, runs routine.RunStore, mergeWaitTimeout time.Duration, notifier ReleaseNotifier, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	if mergeWaitTimeout <= 0 {
		mergeWaitTimeout = 10 * time.Minute
	}
	return &Service{
		repos:             repos,
		accounts:          accounts,
		runs:              runs,
		logger:            logger,
		signal:            make(chan struct{}, 1),
		mergeWaitTimeout:  mergeWaitTimeout,
		mergePollInterval: defaultMergePollInterval,
		notifier:          notifier,
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

// releaseParams is the immutable input persisted in a release run's Run.Params.
type releaseParams struct {
	// Flow discriminates the dev flow ("development": drive an existing MR) from the
	// main flow ("main": create an MR from development into main). An empty Flow is
	// treated as the dev flow so runs created before this field existed keep working.
	Flow string `json:"flow,omitempty"`
	// SourceBranch / TargetBranch are the MR's branches. Dev flow: source is the MR
	// source, target is "development". Main flow: source "development", target "main".
	SourceBranch string `json:"sourceBranch,omitempty"`
	// TargetBranch is captured at creation from the MR (dev flow) or set to the main
	// branch (main flow). The tag suffix and flow branching key off it / Flow.
	TargetBranch string `json:"targetBranch"`
	// Bump is the semver bump mode fed to routine.NextRelease.
	Bump string `json:"bump"`
	// Emojis are the reactions the react step awards.
	Emojis []string `json:"emojis"`
	// RemoveSourceBranch, when true, deletes the source branch on merge. It is a
	// destructive option and DEFAULTS TO false.
	RemoveSourceBranch bool `json:"removeSourceBranch"`
	// MergeWhenPipelineSucceeds, when true, lets GitLab merge as soon as the MR
	// pipeline goes green rather than requiring it green up front. It DEFAULTS TO
	// true.
	MergeWhenPipelineSucceeds bool `json:"mergeWhenPipelineSucceeds"`
}

// releaseState is the mutable accumulator persisted in a release run's Run.State.
// Its JSON fields are served to the SPA (nextTag/lastTag/featCount/fixCount/
// decision) via the run DTO, which emits State as a decoded object.
type releaseState struct {
	LastTag   string `json:"lastTag,omitempty"`
	NextTag   string `json:"nextTag,omitempty"`
	FeatCount int    `json:"featCount"`
	FixCount  int    `json:"fixCount"`
	Decision  string `json:"decision,omitempty"`
	// HeadSHA is the MR diff head captured at the verify step. The merge step pins
	// it via gitlab.MergeOptions.SHA so GitLab rejects the merge if the branch head
	// moved between verification and merge (a TOCTOU guard).
	HeadSHA string `json:"headSHA,omitempty"`
	// MergeTriggered is a durable flag set immediately after a SUCCESSFUL merge
	// call and checkpointed BEFORE the poll loop. It makes the merge idempotent
	// across resumes: once true, the merge step only polls and never re-issues the
	// merge (GitLab rejects a second merge on an already merging/scheduled MR).
	MergeTriggered bool   `json:"mergeTriggered,omitempty"`
	MergeSHA       string `json:"mergeSHA,omitempty"`
	// MRIID is the main flow's created MR. The main flow starts with no MR
	// (run.MRIID == 0) and fills this in at the create_mr step; effectiveMRIID reads
	// run.MRIID for the dev flow and this for the main flow.
	MRIID int `json:"mrIid,omitempty"`
}

// effectiveMRIID returns the merge request IID a release step should act on:
// run.MRIID for the dev flow (the MR exists at creation) or state.MRIID for the
// main flow (the MR is created mid-run at the create_mr step).
func effectiveMRIID(run routine.Run, state releaseState) int {
	if run.MRIID > 0 {
		return run.MRIID
	}
	return state.MRIID
}

// tagSuffix is the prerelease suffix appended to the computed version at the tag
// step: empty for the main flow (a pure release), "-dev" otherwise (the dev flow,
// including runs created before the Flow field existed).
func tagSuffix(flow string) string {
	if flow == flowMain {
		return ""
	}
	return "-dev"
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

// maxEmojis bounds how many reactions a routine will award, so a caller cannot
// force an unbounded number of outbound GitLab calls from a single request.
const maxEmojis = 10

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

	if len(in.Emojis) > maxEmojis {
		return routine.Run{}, fmt.Errorf("routines: too many emojis %d (max %d)", len(in.Emojis), maxEmojis)
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

// --- release routine ---

// ReleaseInput is the request to create a release run. Bump and Emojis are
// optional and fall back to defaults. RemoveSourceBranch and
// MergeWhenPipelineSucceeds are optional merge options passed as pointers so an
// omitted field keeps its default (false and true respectively) rather than
// being forced to the zero value.
type ReleaseInput struct {
	RepoID                    string
	MRIID                     int
	Bump                      string
	Emojis                    []string
	RemoveSourceBranch        *bool
	MergeWhenPipelineSucceeds *bool
}

// CreateRelease validates the repo and MR, applies defaults, and records a
// pending release run with the fixed verify→...→notify ledger, then wakes the
// worker. This slice supports the DEV flow only: the MR must already exist and
// target "development"; any other target is rejected (the main flow is a later
// slice). It returns repo.ErrNotFound for an unknown repo, routine.ErrDuplicateRun
// for a second active run on the same MR, and a validation error otherwise.
func (s *Service) CreateRelease(ctx context.Context, in ReleaseInput) (routine.Run, error) {
	if _, err := s.repos.Get(ctx, in.RepoID); err != nil {
		return routine.Run{}, err
	}
	if in.MRIID <= 0 {
		return routine.Run{}, fmt.Errorf("routines: invalid merge request iid %d (must be positive)", in.MRIID)
	}

	bump := in.Bump
	if bump == "" {
		bump = "minor"
	}
	if !validBumps[bump] {
		return routine.Run{}, fmt.Errorf("routines: invalid bump %q (want major, minor or patch)", bump)
	}
	if len(in.Emojis) > maxEmojis {
		return routine.Run{}, fmt.Errorf("routines: too many emojis %d (max %d)", len(in.Emojis), maxEmojis)
	}
	emojis := in.Emojis
	if len(emojis) == 0 {
		emojis = []string{"thumbsup", "seedling"}
	}

	// Merge-option defaults: RemoveSourceBranch is destructive and defaults to
	// false; MergeWhenPipelineSucceeds defaults to true. A nil pointer means the
	// caller omitted the field, so the default stands.
	removeSourceBranch := false
	if in.RemoveSourceBranch != nil {
		removeSourceBranch = *in.RemoveSourceBranch
	}
	mergeWhenPipelineSucceeds := true
	if in.MergeWhenPipelineSucceeds != nil {
		mergeWhenPipelineSucceeds = *in.MergeWhenPipelineSucceeds
	}

	// Reject a duplicate active run (pending/running/blocked/awaiting) so a second
	// release does not re-drive the same MR.
	runs, err := s.runs.ListByRepo(ctx, in.RepoID)
	if err != nil {
		return routine.Run{}, err
	}
	for _, r := range runs {
		if r.Kind == routine.KindRelease && r.MRIID == in.MRIID && isActiveRun(r.Status) {
			return routine.Run{}, routine.ErrDuplicateRun
		}
	}

	// Read the MR up front to capture (and gate on) its target branch.
	gl, projectID, err := s.gitlabFor(ctx, in.RepoID)
	if err != nil {
		return routine.Run{}, err
	}
	mr, err := gl.GetMergeRequest(ctx, projectID, in.MRIID)
	if err != nil {
		return routine.Run{}, fmt.Errorf("get merge request: %w", err)
	}
	if mr.TargetBranch != devBranch {
		return routine.Run{}, fmt.Errorf("routine: release for target %q is not supported yet (dev flow only)", mr.TargetBranch)
	}

	params, err := json.Marshal(releaseParams{
		Flow:                      flowDevelopment,
		SourceBranch:              mr.SourceBranch,
		TargetBranch:              mr.TargetBranch,
		Bump:                      bump,
		Emojis:                    emojis,
		RemoveSourceBranch:        removeSourceBranch,
		MergeWhenPipelineSucceeds: mergeWhenPipelineSucceeds,
	})
	if err != nil {
		return routine.Run{}, err
	}

	now := time.Now().UTC()
	run := routine.Run{
		ID:     id.New(),
		Kind:   routine.KindRelease,
		RepoID: in.RepoID,
		MRIID:  in.MRIID,
		Status: routine.RunPending,
		Params: params,
		State:  json.RawMessage("{}"),
		Steps: []routine.Step{
			{Name: stepVerify, Status: routine.StepPending},
			{Name: stepReact, Status: routine.StepPending},
			{Name: stepApprove, Status: routine.StepPending},
			{Name: stepComputeTag, Status: routine.StepPending},
			{Name: stepConfirm, Status: routine.StepPending},
			{Name: stepMerge, Status: routine.StepPending},
			{Name: stepTag, Status: routine.StepPending},
			{Name: stepNotify, Status: routine.StepPending},
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

// MainReleaseInput is the request to create a MAIN-flow release run. Unlike the
// dev flow no MR exists yet: the routine creates one from SourceBranch into
// TargetBranch. All fields are optional and fall back to defaults (source
// "development", target "main", bump "minor", the default emojis). The two merge
// options are plain bools here — the HTTP layer resolves its "*bool" defaults
// (RemoveSourceBranch=false, MergeWhenPipelineSucceeds=false) before calling.
type MainReleaseInput struct {
	RepoID                    string
	Bump                      string
	SourceBranch              string
	TargetBranch              string
	Emojis                    []string
	RemoveSourceBranch        bool
	MergeWhenPipelineSucceeds bool
}

// CreateMainRelease validates the repo, applies defaults, and records a pending
// MAIN-flow release run with the compute_tag→create_mr→wait_pipeline→approve→
// react→confirm→merge→tag→notify ledger, then wakes the worker. Like the dev
// flow the main flow PAUSES on the shared confirm gate: state.Decision is left
// empty so stepConfirm returns errAwaitConfirmation and a later Confirm("merge"|
// "wait") drives it. The run starts with run.MRIID=0 (the create_mr step fills
// state.MRIID). It returns repo.ErrNotFound for an unknown repo,
// routine.ErrDuplicateRun for a second active main run targeting the same branch,
// and a validation error otherwise.
func (s *Service) CreateMainRelease(ctx context.Context, in MainReleaseInput) (routine.Run, error) {
	if _, err := s.repos.Get(ctx, in.RepoID); err != nil {
		return routine.Run{}, err
	}

	sourceBranch := in.SourceBranch
	if sourceBranch == "" {
		sourceBranch = devBranch
	}
	targetBranch := in.TargetBranch
	if targetBranch == "" {
		targetBranch = mainBranch
	}
	// A release must move commits between two DIFFERENT branches; a same-branch MR
	// is impossible, so reject it up front (the HTTP layer maps this to 400).
	if sourceBranch == targetBranch {
		return routine.Run{}, fmt.Errorf("routines: source branch %q must differ from target branch %q", sourceBranch, targetBranch)
	}
	bump := in.Bump
	if bump == "" {
		bump = "minor"
	}
	if !validBumps[bump] {
		return routine.Run{}, fmt.Errorf("routines: invalid bump %q (want major, minor or patch)", bump)
	}
	if len(in.Emojis) > maxEmojis {
		return routine.Run{}, fmt.Errorf("routines: too many emojis %d (max %d)", len(in.Emojis), maxEmojis)
	}
	emojis := in.Emojis
	if len(emojis) == 0 {
		emojis = []string{"thumbsup", "seedling"}
	}

	// Reject a duplicate active MAIN run for the same repo+target branch. The main
	// flow has no MR at creation, so dedupe on flow+targetBranch rather than MR IID.
	//
	// TODO: this scan is check-then-act — two HTTP creations racing here could both
	// pass the check and create two runs. The single worker limits the blast radius
	// to two sequential releases (never a concurrent double-merge), so no store-level
	// uniqueness constraint is added now.
	runs, err := s.runs.ListByRepo(ctx, in.RepoID)
	if err != nil {
		return routine.Run{}, err
	}
	for _, r := range runs {
		if r.Kind != routine.KindRelease || !isActiveRun(r.Status) {
			continue
		}
		var p releaseParams
		if len(r.Params) > 0 {
			_ = json.Unmarshal(r.Params, &p)
		}
		if p.Flow == flowMain && p.TargetBranch == targetBranch {
			return routine.Run{}, routine.ErrDuplicateRun
		}
	}

	params, err := json.Marshal(releaseParams{
		Flow:                      flowMain,
		SourceBranch:              sourceBranch,
		TargetBranch:              targetBranch,
		Bump:                      bump,
		Emojis:                    emojis,
		RemoveSourceBranch:        in.RemoveSourceBranch,
		MergeWhenPipelineSucceeds: in.MergeWhenPipelineSucceeds,
	})
	if err != nil {
		return routine.Run{}, err
	}

	now := time.Now().UTC()
	run := routine.Run{
		ID:     id.New(),
		Kind:   routine.KindRelease,
		RepoID: in.RepoID,
		MRIID:  0,
		Status: routine.RunPending,
		Params: params,
		// State starts empty: like the dev flow, the confirm step pauses the run at
		// awaiting_confirmation until Confirm records a merge decision.
		State: json.RawMessage("{}"),
		Steps: []routine.Step{
			{Name: stepComputeTag, Status: routine.StepPending},
			{Name: stepCreateMR, Status: routine.StepPending},
			{Name: stepWaitPipeline, Status: routine.StepPending},
			{Name: stepApprove, Status: routine.StepPending},
			{Name: stepReact, Status: routine.StepPending},
			{Name: stepConfirm, Status: routine.StepPending},
			{Name: stepMerge, Status: routine.StepPending},
			{Name: stepTag, Status: routine.StepPending},
			{Name: stepNotify, Status: routine.StepPending},
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

// Confirm resolves the release confirmation gate: it records the merge decision
// ("merge" or "wait") on the run's state and flips it back to pending so the
// worker re-enters the confirm step. The run must be awaiting_confirmation
// (otherwise routine.ErrNotAwaitingConfirmation); an unknown decision is a
// validation error.
func (s *Service) Confirm(ctx context.Context, runID, decision string) (routine.Run, error) {
	if decision != "merge" && decision != "wait" {
		return routine.Run{}, fmt.Errorf("%w: got %q", routine.ErrInvalidConfirmationDecision, decision)
	}
	run, err := s.runs.Get(ctx, runID)
	if err != nil {
		return routine.Run{}, err
	}
	if run.Status != routine.RunAwaitingConfirmation {
		return routine.Run{}, routine.ErrNotAwaitingConfirmation
	}

	var state releaseState
	if len(run.State) > 0 {
		_ = json.Unmarshal(run.State, &state)
	}
	state.Decision = decision
	run.State = marshalReleaseState(state)
	run.Status = routine.RunPending
	run.LastError = ""
	if err := s.runs.Save(ctx, run); err != nil {
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

// execute dispatches a run to the driver for its kind.
func (s *Service) execute(ctx context.Context, run routine.Run) error {
	switch run.Kind {
	case routine.KindRelease:
		return s.executeRelease(ctx, run)
	default:
		return s.executeApproveAndTag(ctx, run)
	}
}

// executeApproveAndTag drives an approve_and_tag run's steps to completion,
// checkpointing after each. A failed step blocks the run (an expected pause,
// returned as nil so the worker keeps going); a genuine persistence failure is
// returned as an error.
func (s *Service) executeApproveAndTag(ctx context.Context, run routine.Run) error {
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

// executeRelease drives a release run's steps to completion, checkpointing after
// each. It mirrors executeApproveAndTag but adds the confirmation gate: when the
// confirm step returns errAwaitConfirmation the run pauses in
// awaiting_confirmation (not blocked, not failed) so a later Confirm re-enters
// the same step. A failed step blocks the run; a persistence failure is returned.
func (s *Service) executeRelease(ctx context.Context, run routine.Run) error {
	gl, projectID, err := s.gitlabFor(ctx, run.RepoID)
	if err != nil {
		return s.block(ctx, run, fmt.Errorf("gitlab client: %w", err))
	}

	var params releaseParams
	if err := json.Unmarshal(run.Params, &params); err != nil {
		return s.block(ctx, run, fmt.Errorf("decode params: %w", err))
	}

	var state releaseState
	if len(run.State) > 0 {
		_ = json.Unmarshal(run.State, &state)
	}

	// checkpoint persists a mid-step decision (the computed tag, the merge SHA)
	// while the step is still running, so a crash cannot recompute it on resume.
	checkpoint := func(st *releaseState) error {
		run.State = marshalReleaseState(*st)
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
		run.State = marshalReleaseState(state)
		if err := s.runs.Save(ctx, run); err != nil {
			return fmt.Errorf("routines: save run %s: %w", run.ID, err)
		}

		detail, herr := s.runReleaseStep(ctx, gl, projectID, run, step.Name, params, &state, checkpoint)
		step.UpdatedAt = time.Now().UTC()
		run.State = marshalReleaseState(state)

		// The confirmation gate is a pause, not a failure: leave the confirm step
		// non-terminal (StepRunning) so the resume that Confirm triggers re-enters
		// it, and flip the run to awaiting_confirmation.
		if errors.Is(herr, errAwaitConfirmation) {
			run.Status = routine.RunAwaitingConfirmation
			if err := s.runs.Save(ctx, run); err != nil {
				return fmt.Errorf("routines: save run %s: %w", run.ID, err)
			}
			s.logger.Printf("routines: run %s awaiting confirmation at step %q", run.ID, step.Name)
			return nil
		}

		// Nothing-to-release is a clean stop, not a failure: mark the compute step
		// done with an explanatory detail and finish the run WITHOUT opening an MR or
		// running the remaining steps.
		if errors.Is(herr, errNothingToRelease) {
			base := state.LastTag
			if base == "" {
				base = "the last release"
			}
			step.Status = routine.StepDone
			step.Detail = fmt.Sprintf("no releasable commits since %s; nothing to release", base)
			run.Status = routine.RunDone
			run.LastError = ""
			run.State = marshalReleaseState(state)
			if err := s.runs.Save(ctx, run); err != nil {
				return fmt.Errorf("routines: save run %s: %w", run.ID, err)
			}
			s.logger.Printf("routines: run %s finished: %s", run.ID, step.Detail)
			return nil
		}

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
	run.State = marshalReleaseState(state)
	if err := s.runs.Save(ctx, run); err != nil {
		return fmt.Errorf("routines: save run %s: %w", run.ID, err)
	}
	return nil
}

// runReleaseStep executes one release step (check-then-act / idempotent) and
// returns a short human-readable detail on success. It may mutate state.
func (s *Service) runReleaseStep(ctx context.Context, gl *gitlab.Client, projectID string, run routine.Run, name string, params releaseParams, state *releaseState, checkpoint func(*releaseState) error) (string, error) {
	// Both flows key their MR-scoped steps off the effective IID: the dev flow's
	// pre-existing run.MRIID, or the main flow's state.MRIID (set at create_mr).
	mrIID := effectiveMRIID(run, *state)
	switch name {
	case stepVerify:
		// The MR must be mergeable: no conflicts, and any latest pipeline must be
		// green. A not-yet-finished or failed pipeline blocks (an expected pause):
		// the user pushes a fix / waits for green, then resumes.
		//
		// Asymmetry with the merge step (intentional): verify GATES ENTRY — it
		// requires the pipeline to already be green and, if not, blocks and demands
		// a manual Resume. The merge step, by contrast, OWNS THE LONG WAIT — it
		// auto-polls the MR to the "merged" state. So verify does not poll (it fails
		// fast and hands control back to a human), while merge does (it waits for
		// GitLab to complete the merge without human involvement).
		mr, err := gl.GetMergeRequest(ctx, projectID, mrIID)
		if err != nil {
			return "", fmt.Errorf("get merge request: %w", err)
		}
		if mr.HasConflicts || mr.MergeStatus == "cannot_be_merged" {
			return "", fmt.Errorf("merge request !%d has conflicts; resolve and resume", mrIID)
		}
		pipelines, err := gl.MergeRequestPipelines(ctx, projectID, mrIID)
		if err != nil {
			return "", fmt.Errorf("list merge request pipelines: %w", err)
		}
		if latest, ok := latestPipeline(pipelines); ok {
			switch latest.Status {
			case "success":
				// green — proceed
			case "running", "pending", "created", "preparing", "scheduled", "waiting_for_resource", "manual":
				return "", fmt.Errorf("pipeline not finished (status %q); resume when green", latest.Status)
			case "failed", "canceled", "skipped":
				return "", fmt.Errorf("pipeline failed (status %q); fix it and resume", latest.Status)
			default:
				return "", fmt.Errorf("pipeline not finished (status %q); resume when green", latest.Status)
			}
		}
		// Pin the verified head so the merge step can reject a merge if the branch
		// moved between here and there (the confirmation gate can sit indefinitely).
		state.HeadSHA = mr.SHA
		if err := checkpoint(state); err != nil {
			return "", fmt.Errorf("checkpoint verified head: %w", err)
		}
		return "verified", nil

	case stepReact:
		for _, emoji := range params.Emojis {
			// AwardEmoji is idempotent: a duplicate award is swallowed.
			if err := gl.AwardEmoji(ctx, projectID, mrIID, emoji); err != nil {
				return "", fmt.Errorf("award emoji %q: %w", emoji, err)
			}
		}
		return "reacted", nil

	case stepApprove:
		// ApproveMergeRequest is idempotent (already-approved is swallowed).
		if err := gl.ApproveMergeRequest(ctx, projectID, mrIID); err != nil {
			return "", fmt.Errorf("approve merge request: %w", err)
		}
		return "approved", nil

	case stepComputeTag:
		if params.Flow == flowMain {
			return s.runComputeTagMain(ctx, gl, projectID, params, state, checkpoint)
		}
		// The next version is a durable decision: compute it once and persist it
		// before any side effect, so a resume reuses it rather than recomputing a
		// different (wrongly bumped) value.
		if state.NextTag == "" {
			tags, err := gl.ListTags(ctx, projectID)
			if err != nil {
				return "", fmt.Errorf("list tags: %w", err)
			}
			existing := make([]string, 0, len(tags))
			for _, t := range tags {
				existing = append(existing, t.Name)
			}
			lastTag := routine.HighestSemver(existing)

			commits, err := gl.MergeRequestCommits(ctx, projectID, mrIID)
			if err != nil {
				return "", fmt.Errorf("get merge request commits: %w", err)
			}
			// GitLab returns commits newest first; NextRelease wants oldest first.
			subjects := make([]string, len(commits))
			for i, c := range commits {
				subjects[len(commits)-1-i] = c.Title
			}

			next, counts, err := routine.NextRelease(lastTag, subjects, routine.BumpMode(params.Bump))
			if err != nil {
				return "", fmt.Errorf("compute next release: %w", err)
			}
			state.LastTag = lastTag
			state.NextTag = next
			state.FeatCount = counts.Feat
			state.FixCount = counts.Fix
			if err := checkpoint(state); err != nil {
				return "", fmt.Errorf("checkpoint computed tag: %w", err)
			}
		}
		return fmt.Sprintf("next release %s (feat %d, fix %d)", state.NextTag, state.FeatCount, state.FixCount), nil

	case stepCreateMR:
		return s.runCreateMRStep(ctx, gl, projectID, params, state, checkpoint)

	case stepWaitPipeline:
		return s.runWaitPipelineStep(ctx, gl, projectID, mrIID, state, checkpoint)

	case stepConfirm:
		// Interactive gate: pause until Confirm records a decision.
		if state.Decision == "" {
			return "", errAwaitConfirmation
		}
		return "decision: " + state.Decision, nil

	case stepMerge:
		return s.runMergeStep(ctx, gl, projectID, mrIID, params, state, checkpoint)

	case stepTag:
		if state.MergeSHA == "" {
			return "", fmt.Errorf("merge request !%d has no merge commit SHA to tag", mrIID)
		}
		// The tag name is the computed next version plus a flow-dependent suffix:
		// "-dev" for the dev flow, empty (a pure release) for the main flow.
		finalName := state.NextTag + tagSuffix(params.Flow)

		tags, err := gl.ListTags(ctx, projectID)
		if err != nil {
			return "", fmt.Errorf("list tags: %w", err)
		}
		existing := make([]string, 0, len(tags))
		for _, t := range tags {
			existing = append(existing, t.Name)
		}
		if tagExists(existing, finalName) {
			return fmt.Sprintf("tag %s already exists", finalName), nil
		}
		if err := gl.CreateTag(ctx, projectID, finalName, state.MergeSHA, ""); err != nil {
			return "", fmt.Errorf("create tag %s: %w", finalName, err)
		}
		return fmt.Sprintf("created tag %s", finalName), nil

	case stepNotify:
		// Best-effort: a notify failure must NEVER fail the run.
		//
		// Build an informative plain-text summary from the run's state: the created
		// tag, the repo, the flow, and the merged MR. The repo name is best-effort —
		// fall back to the repo id if the lookup fails, since notification must never
		// hold up (or fail) a finished release.
		flow := params.Flow
		if flow == "" {
			flow = flowDevelopment
		}
		repoName := run.RepoID
		if rp, err := s.repos.Get(ctx, run.RepoID); err == nil && rp.Name != "" {
			repoName = rp.Name
		}
		summary := fmt.Sprintf("Release %s completed for %s (%s flow), MR !%d merged", state.NextTag+tagSuffix(params.Flow), repoName, flow, mrIID)
		if s.notifier == nil {
			s.logger.Printf("routines: %s (no notifier configured)", summary)
			return "notify skipped (no notifier)", nil
		}
		if err := s.notifier.Notify(ctx, summary); err != nil {
			s.logger.Printf("routines: notify failed for MR !%d: %v", mrIID, err)
			return "notify failed (ignored)", nil
		}
		return "notified", nil

	default:
		return "", fmt.Errorf("unknown step %q", name)
	}
}

// runMergeStep triggers the merge exactly once (only for the "merge" decision)
// and then polls until the MR reaches the "merged" state, capturing the merge
// commit SHA. It honors ctx and is bounded by s.mergeWaitTimeout; on timeout it
// returns an error so the run blocks and a resume re-polls. A single worker means
// a long merge-wait serializes other runs — acceptable for now.
//
// Idempotency across resumes is durable, not state-derived: a poll timeout blocks
// the run, and on resume this step re-enters. Re-issuing MergeMergeRequest on an
// MR that is already merging/scheduled makes GitLab reject it (405/406), which
// would wrongly re-block a merge that was about to succeed. The MergeTriggered
// flag — set and checkpointed immediately after the successful merge call, BEFORE
// the poll loop — guarantees the merge is issued at most once: after a resume the
// flag is already true, so the step only polls and never re-merges.
func (s *Service) runMergeStep(ctx context.Context, gl *gitlab.Client, projectID string, mrIID int, params releaseParams, state *releaseState, checkpoint func(*releaseState) error) (string, error) {
	// Trigger the merge at most once, and only for the "merge" decision (a "wait"
	// decision never merges — it only polls for an out-of-band merge below).
	if state.Decision == "merge" && !state.MergeTriggered {
		// TOCTOU guard: the confirmation gate can sit indefinitely between verify
		// and merge. Re-fetch the MR right before merging and refuse to merge one
		// that became unmergeable since verification.
		mr, err := gl.GetMergeRequest(ctx, projectID, mrIID)
		if err != nil {
			return "", fmt.Errorf("get merge request: %w", err)
		}
		if mr.State != "merged" {
			if mr.HasConflicts || mr.MergeStatus == "cannot_be_merged" {
				return "", fmt.Errorf("merge request !%d became unmergeable since verification; resolve and resume", mrIID)
			}
			// Pin the head captured at verify (state.HeadSHA): GitLab rejects the
			// merge if the branch head moved since (a stale-head 409), which blocks
			// the run with a clear "changed since verification; re-run" message.
			if _, err := gl.MergeMergeRequest(ctx, projectID, mrIID, gitlab.MergeOptions{
				MergeWhenPipelineSucceeds: params.MergeWhenPipelineSucceeds,
				RemoveSourceBranch:        params.RemoveSourceBranch,
				SHA:                       state.HeadSHA,
			}); err != nil {
				return "", fmt.Errorf("merge merge request (MR changed since verification; re-run if the branch head moved): %w", err)
			}
		}
		// Record the trigger durably BEFORE polling so any resume only polls and
		// never re-issues the merge.
		state.MergeTriggered = true
		if err := checkpoint(state); err != nil {
			return "", fmt.Errorf("checkpoint merge triggered: %w", err)
		}
	}

	deadline := time.Now().Add(s.mergeWaitTimeout)
	for {
		mr, err := gl.GetMergeRequest(ctx, projectID, mrIID)
		if err != nil {
			return "", fmt.Errorf("get merge request: %w", err)
		}
		if mr.State == "merged" {
			sha := mr.MergeCommitSHA
			if sha == "" {
				sha = mr.SquashCommitSHA
			}
			if sha == "" {
				sha = mr.SHA
			}
			if sha == "" {
				return "", fmt.Errorf("merge request !%d is merged but exposes no commit SHA to tag", mrIID)
			}
			state.MergeSHA = sha
			if err := checkpoint(state); err != nil {
				return "", fmt.Errorf("checkpoint merge sha: %w", err)
			}
			return "merged " + sha, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("merge request !%d still waiting for merge; resume to keep waiting", mrIID)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(s.mergePollInterval):
		}
	}
}

// runComputeTagMain computes the next MAIN release version. Its base is the
// highest PURE RELEASE tag (prerelease/"-dev" tags are ignored, so a 1.7.2-dev
// never becomes the base for a main release). The commits counted are those on the
// source branch (development) since that last release tag — or since the target
// branch (main) when there is no release tag yet. Like the dev variant the result
// is a durable decision: computed once and checkpointed before any side effect.
func (s *Service) runComputeTagMain(ctx context.Context, gl *gitlab.Client, projectID string, params releaseParams, state *releaseState, checkpoint func(*releaseState) error) (string, error) {
	if state.NextTag == "" {
		tags, err := gl.ListTags(ctx, projectID)
		if err != nil {
			return "", fmt.Errorf("list tags: %w", err)
		}
		existing := make([]string, 0, len(tags))
		for _, t := range tags {
			existing = append(existing, t.Name)
		}
		// Ignore prerelease/"-dev" tags: a main release bases off pure X.Y.Z only.
		lastTag := routine.HighestReleaseSemver(existing)

		// Count commits reachable from the source branch (development) but not from
		// the base ref: the last release tag if present, else the target branch.
		from := lastTag
		if from == "" {
			from = params.TargetBranch
		}
		cmp, err := gl.CompareRefs(ctx, projectID, from, params.SourceBranch)
		if err != nil {
			return "", fmt.Errorf("compare refs %s...%s: %w", from, params.SourceBranch, err)
		}
		// CompareRefs returns commits oldest-first, which is the order NextRelease
		// wants, so the titles are used as-is (no reverse, unlike the dev variant).
		subjects := make([]string, len(cmp.Commits))
		for i, c := range cmp.Commits {
			subjects[i] = c.Title
		}

		next, counts, err := routine.NextRelease(lastTag, subjects, routine.BumpMode(params.Bump))
		if err != nil {
			return "", fmt.Errorf("compute next release: %w", err)
		}
		// Nothing to release: no feat/fix commits since the base means next == lastTag.
		// Persist the base so executeRelease can name it, then stop the run without
		// opening an MR (opening an empty development→main MR would be noise).
		if counts.Feat == 0 && counts.Fix == 0 {
			state.LastTag = lastTag
			if err := checkpoint(state); err != nil {
				return "", fmt.Errorf("checkpoint computed tag: %w", err)
			}
			return "", errNothingToRelease
		}
		state.LastTag = lastTag // "" when there is no previous release (UI shows "no previous")
		state.NextTag = next
		state.FeatCount = counts.Feat
		state.FixCount = counts.Fix
		if err := checkpoint(state); err != nil {
			return "", fmt.Errorf("checkpoint computed tag: %w", err)
		}
	}
	return fmt.Sprintf("next release %s (feat %d, fix %d)", state.NextTag, state.FeatCount, state.FixCount), nil
}

// runCreateMRStep creates the main flow's development→main MR, or REUSES the one
// this routine opened on a prior attempt (check-then-act for resume idempotency:
// a crash after CreateMergeRequest but before the checkpoint would otherwise open
// a duplicate). The resulting IID is persisted in state.MRIID so the later
// MR-scoped steps act on it.
//
// Anti-hijack: it only ever reuses an open source→target MR whose title carries
// the mainReleaseMarker prefix. An open source→target MR WITHOUT that marker is a
// human's unrelated MR — adopting it would merge and tag work this routine never
// reviewed — so its presence BLOCKS the run instead. The branch filter is applied
// server-side (OpenMergeRequestsForBranches) rather than over an unpaginated list.
func (s *Service) runCreateMRStep(ctx context.Context, gl *gitlab.Client, projectID string, params releaseParams, state *releaseState, checkpoint func(*releaseState) error) (string, error) {
	// Reuse: if state.MRIID is already set (resume) nothing to do; otherwise inspect
	// the open source→target MRs before creating a new one.
	if state.MRIID == 0 {
		open, err := gl.OpenMergeRequestsForBranches(ctx, projectID, params.SourceBranch, params.TargetBranch)
		if err != nil {
			return "", fmt.Errorf("list open merge requests: %w", err)
		}
		var marked []gitlab.MergeRequest
		for _, mr := range open {
			// The server already filtered by branch pair; guard defensively anyway.
			if mr.SourceBranch != params.SourceBranch || mr.TargetBranch != params.TargetBranch {
				continue
			}
			if !strings.HasPrefix(mr.Title, mainReleaseMarker) {
				// A human opened an unrelated development→main MR: refuse to adopt it.
				return "", fmt.Errorf("an unrelated %s->%s merge request is already open; close or merge it before running a release", params.SourceBranch, params.TargetBranch)
			}
			marked = append(marked, mr)
		}
		if len(marked) > 0 {
			// Resume idempotency: reuse the MR this routine created. Pick the lowest
			// IID deterministically if somehow more than one marked MR is open.
			lowest := marked[0]
			for _, mr := range marked[1:] {
				if mr.IID < lowest.IID {
					lowest = mr
				}
			}
			if len(marked) > 1 {
				s.logger.Printf("routines: found %d open %s->%s release merge requests; reusing lowest IID !%d", len(marked), params.SourceBranch, params.TargetBranch, lowest.IID)
			}
			state.MRIID = lowest.IID
		}
	}
	if state.MRIID == 0 {
		title := fmt.Sprintf("%s %s TAG: %s", mainReleaseMarker, time.Now().Format("02.01.2006"), state.NextTag)
		description := fmt.Sprintf("Commits: feat %d, fix %d", state.FeatCount, state.FixCount)
		mr, err := gl.CreateMergeRequest(ctx, projectID, params.SourceBranch, params.TargetBranch, title, description)
		if err != nil {
			return "", fmt.Errorf("create merge request: %w", err)
		}
		state.MRIID = mr.IID
	}
	if err := checkpoint(state); err != nil {
		return "", fmt.Errorf("checkpoint created MR: %w", err)
	}
	return fmt.Sprintf("merge request !%d ready (%s -> %s)", state.MRIID, params.SourceBranch, params.TargetBranch), nil
}

// runWaitPipelineStep re-fetches the created MR and blocks on conflicts, then
// polls the latest pipeline until it succeeds. A failed/canceled pipeline blocks
// (an expected pause: fix and resume); no pipeline at all is treated as OK. It is
// bounded by s.mergeWaitTimeout and honors ctx; on timeout it blocks so a resume
// keeps waiting.
//
// The main flow has no verify step, so this is where the merge head is pinned:
// right before returning success it records state.HeadSHA = mr.SHA (the MR's
// current diff head) so the merge step's SHA pin rejects a merge if development
// moved after the pipeline went green (the confirm gate can sit indefinitely).
func (s *Service) runWaitPipelineStep(ctx context.Context, gl *gitlab.Client, projectID string, mrIID int, state *releaseState, checkpoint func(*releaseState) error) (string, error) {
	mr, err := gl.GetMergeRequest(ctx, projectID, mrIID)
	if err != nil {
		return "", fmt.Errorf("get merge request: %w", err)
	}
	if mr.HasConflicts || mr.MergeStatus == "cannot_be_merged" {
		return "", fmt.Errorf("created merge request !%d has conflicts; resolve and resume", mrIID)
	}
	// pinHead records the verified diff head so runMergeStep can SHA-pin the merge.
	pinHead := func() error {
		state.HeadSHA = mr.SHA
		return checkpoint(state)
	}
	deadline := time.Now().Add(s.mergeWaitTimeout)
	for {
		pipelines, err := gl.MergeRequestPipelines(ctx, projectID, mrIID)
		if err != nil {
			return "", fmt.Errorf("list merge request pipelines: %w", err)
		}
		latest, ok := latestPipeline(pipelines)
		if !ok {
			// No pipeline attached: nothing to wait for, proceed.
			if err := pinHead(); err != nil {
				return "", fmt.Errorf("checkpoint verified head: %w", err)
			}
			return "no pipeline; proceeding", nil
		}
		switch latest.Status {
		case "success":
			if err := pinHead(); err != nil {
				return "", fmt.Errorf("checkpoint verified head: %w", err)
			}
			return "pipeline succeeded", nil
		case "failed", "canceled", "skipped":
			return "", fmt.Errorf("pipeline failed (status %q); fix it and resume", latest.Status)
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("still waiting for pipeline (status %q); resume to keep waiting", latest.Status)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(s.mergePollInterval):
		}
	}
}

// latestPipeline returns the most recent pipeline (highest ID) among the ones
// attached to an MR, reporting false when there are none.
func latestPipeline(pipelines []gitlab.Pipeline) (gitlab.Pipeline, bool) {
	if len(pipelines) == 0 {
		return gitlab.Pipeline{}, false
	}
	latest := pipelines[0]
	for _, p := range pipelines[1:] {
		if p.ID > latest.ID {
			latest = p
		}
	}
	return latest, true
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

// marshalReleaseState encodes the release accumulator, falling back to an empty
// object so Run.State is always valid JSON.
func marshalReleaseState(state releaseState) json.RawMessage {
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
	case routine.RunPending, routine.RunRunning, routine.RunBlocked, routine.RunAwaitingConfirmation:
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
