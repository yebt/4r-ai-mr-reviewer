package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/app/routines"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
)

// repoPreflight reports which merge-request routine actions the repo's token and
// access level permit. Unknown repo → 404; upstream GitLab failure → 502.
func (s *Server) repoPreflight(w http.ResponseWriter, r *http.Request) {
	pf, err := s.routines.Preflight(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, toPreflight(pf))
}

// createApproveAndTagRoutine creates an approve_and_tag run for a repo's MR.
func (s *Server) createApproveAndTagRoutine(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MRIID   int      `json:"mrIid"`
		Emojis  []string `json:"emojis"`
		Comment string   `json:"comment"`
		Bump    string   `json:"bump"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.routines.CreateApproveAndTag(r.Context(), routines.ApproveAndTagInput{
		RepoID:  r.PathValue("id"),
		MRIID:   in.MRIID,
		Emojis:  in.Emojis,
		Comment: in.Comment,
		Bump:    in.Bump,
	})
	if err != nil {
		if errors.Is(err, routine.ErrDuplicateRun) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toRun(run))
}

// createReleaseRoutine creates a release run for a repo's MR (dev flow only).
func (s *Server) createReleaseRoutine(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MRIID  int      `json:"mrIid"`
		Bump   string   `json:"bump"`
		Emojis []string `json:"emojis"`
		// Optional merge options. When omitted (nil), the service applies its
		// defaults: RemoveSourceBranch=false, MergeWhenPipelineSucceeds=true.
		RemoveSourceBranch        *bool `json:"removeSourceBranch"`
		MergeWhenPipelineSucceeds *bool `json:"mergeWhenPipelineSucceeds"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.routines.CreateRelease(r.Context(), routines.ReleaseInput{
		RepoID:                    r.PathValue("id"),
		MRIID:                     in.MRIID,
		Bump:                      in.Bump,
		Emojis:                    in.Emojis,
		RemoveSourceBranch:        in.RemoveSourceBranch,
		MergeWhenPipelineSucceeds: in.MergeWhenPipelineSucceeds,
	})
	if err != nil {
		if errors.Is(err, routine.ErrDuplicateRun) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toRun(run))
}

// createMainReleaseRoutine creates a MAIN-flow release run for a repo: it creates
// the development→main MR itself, waits for its pipeline to go green, then pauses
// on the shared confirmation gate before merging. No MR need exist yet.
func (s *Server) createMainReleaseRoutine(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Bump         string   `json:"bump"`
		IncludeDev   bool     `json:"includeDev"`
		SourceBranch string   `json:"sourceBranch"`
		TargetBranch string   `json:"targetBranch"`
		Emojis       []string `json:"emojis"`
		// Optional merge options. When omitted (nil), the defaults apply:
		// RemoveSourceBranch=false, MergeWhenPipelineSucceeds=false (the main flow
		// already gates on a green pipeline in wait_pipeline, so an immediate
		// all-or-nothing merge of the pinned head is safer than scheduling an async
		// MWPS merge).
		RemoveSourceBranch        *bool `json:"removeSourceBranch"`
		MergeWhenPipelineSucceeds *bool `json:"mergeWhenPipelineSucceeds"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	removeSourceBranch := false
	if in.RemoveSourceBranch != nil {
		removeSourceBranch = *in.RemoveSourceBranch
	}
	mergeWhenPipelineSucceeds := false
	if in.MergeWhenPipelineSucceeds != nil {
		mergeWhenPipelineSucceeds = *in.MergeWhenPipelineSucceeds
	}
	run, err := s.routines.CreateMainRelease(r.Context(), routines.MainReleaseInput{
		RepoID:                    r.PathValue("id"),
		Bump:                      in.Bump,
		IncludeDev:                in.IncludeDev,
		SourceBranch:              in.SourceBranch,
		TargetBranch:              in.TargetBranch,
		Emojis:                    in.Emojis,
		RemoveSourceBranch:        removeSourceBranch,
		MergeWhenPipelineSucceeds: mergeWhenPipelineSucceeds,
	})
	if err != nil {
		if errors.Is(err, routine.ErrDuplicateRun) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toRun(run))
}

// confirmRoutine resolves a release run's confirmation gate. An invalid decision
// → 400; a non-awaiting run → 409; an unknown run → 404 (and any unexpected error
// → 500) via writeErr, mirroring resumeRoutine's routing.
func (s *Server) confirmRoutine(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision string `json:"decision"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.routines.Confirm(r.Context(), r.PathValue("id"), in.Decision)
	if err != nil {
		switch {
		case errors.Is(err, routine.ErrInvalidConfirmationDecision):
			writeErr(w, err, http.StatusBadRequest)
		case errors.Is(err, routine.ErrNotAwaitingConfirmation):
			writeErr(w, err, http.StatusConflict)
		case errors.Is(err, routine.ErrRunFinalized):
			// The run was cancelled/finalized concurrently with this confirm.
			writeErr(w, err, http.StatusConflict)
		default:
			writeErr(w, err, http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// getRoutine returns a single routine run. Unknown run → 404.
func (s *Server) getRoutine(w http.ResponseWriter, r *http.Request) {
	run, err := s.routines.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, routine.ErrRunNotFound) {
			writeErr(w, err, http.StatusNotFound)
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// listRoutines returns a repo's routine runs, newest first.
func (s *Server) listRoutines(w http.ResponseWriter, r *http.Request) {
	list := s.routines.ListByRepo
	if q := r.URL.Query().Get("archived"); q == "1" || q == "true" {
		list = s.routines.ListArchivedByRepo
	}
	runs, err := list(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]routineRunResp, 0, len(runs))
	for _, run := range runs {
		out = append(out, toRun(run))
	}
	writeJSON(w, http.StatusOK, out)
}

// listRecentRoutines returns recent routine runs across all repos, newest first.
// An optional ?limit=N caps the result (the service clamps it to a sane range).
// Each item carries a best-effort repoName resolved via the repos service so the
// global list can show which repo a run belongs to; an unresolvable repo yields
// an empty repoName rather than failing the whole list.
func (s *Server) listRecentRoutines(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			limit = n
		}
	}
	list := s.routines.ListRecent
	if q := r.URL.Query().Get("archived"); q == "1" || q == "true" {
		list = s.routines.ListRecentArchived
	}
	runs, err := list(r.Context(), limit)
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	// Cache repo-name lookups so several runs from the same repo cost one lookup.
	names := make(map[string]string)
	out := make([]routineRunResp, 0, len(runs))
	for _, run := range runs {
		dto := toRun(run)
		name, ok := names[run.RepoID]
		if !ok {
			if rp, gerr := s.repos.Get(r.Context(), run.RepoID); gerr == nil {
				name = rp.Name
			}
			names[run.RepoID] = name
		}
		dto.RepoName = name
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

// listRepoBranches returns the repo's branch names for the release branch picker.
func (s *Server) listRepoBranches(w http.ResponseWriter, r *http.Request) {
	names, err := s.routines.ListBranches(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, names)
}

// previewRoutineTag computes the next release tag for a repo without creating a
// run (a dry-run), so the release modal can show the exact tag before launch.
// Query: flow (main|dev), bump, mrIid (dev), source/target (main).
func (s *Server) previewRoutineTag(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mrIID := 0
	if v := q.Get("mrIid"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			mrIID = n
		}
	}
	res, err := s.routines.PreviewNextTag(r.Context(), routines.PreviewInput{
		RepoID:       r.PathValue("id"),
		Flow:         q.Get("flow"),
		MRIID:        mrIID,
		Bump:         q.Get("bump"),
		IncludeDev:   q.Get("includeDev") == "1" || q.Get("includeDev") == "true",
		SourceBranch: q.Get("source"),
		TargetBranch: q.Get("target"),
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, previewTagResp{
		NextTag: res.NextTag, LastTag: res.LastTag, FeatCount: res.FeatCount, FixCount: res.FixCount,
	})
}

// deleteRoutine removes a routine run. A running run cannot be deleted → 409;
// an unknown run → 404; success → 204.
func (s *Server) deleteRoutine(w http.ResponseWriter, r *http.Request) {
	if err := s.routines.Delete(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, routine.ErrRunActive) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// archiveRoutine soft-hides a finished run from the active list. A non-terminal
// run → 409; an unknown run → 404; success → 200.
func (s *Server) archiveRoutine(w http.ResponseWriter, r *http.Request) {
	if err := s.routines.Archive(r.Context(), r.PathValue("id")); err != nil {
		switch {
		case errors.Is(err, routine.ErrRunActive):
			writeErr(w, err, http.StatusConflict)
		case errors.Is(err, routine.ErrRunNotFound):
			writeErr(w, err, http.StatusNotFound)
		default:
			writeErr(w, err, http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

// unarchiveRoutine restores an archived run to the active list. Unknown run → 404.
func (s *Server) unarchiveRoutine(w http.ResponseWriter, r *http.Request) {
	if err := s.routines.Unarchive(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, routine.ErrRunNotFound) {
			writeErr(w, err, http.StatusNotFound)
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unarchived"})
}

// resumeRoutine re-queues a blocked run. A non-blocked run → 409.
func (s *Server) resumeRoutine(w http.ResponseWriter, r *http.Request) {
	run, err := s.routines.Resume(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, routine.ErrNotResumable) || errors.Is(err, routine.ErrRunFinalized) {
			// ErrRunFinalized: the run was cancelled/finalized concurrently.
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// skipRoutine marks a blocked run's failed step as skipped and re-queues it. A
// non-blocked run or an essential (non-skippable) failed step → 409; an unknown
// run → 404; any other error → 500.
func (s *Server) skipRoutine(w http.ResponseWriter, r *http.Request) {
	run, err := s.routines.Skip(r.Context(), r.PathValue("id"))
	if err != nil {
		switch {
		case errors.Is(err, routine.ErrNotResumable),
			errors.Is(err, routine.ErrStepNotSkippable),
			errors.Is(err, routine.ErrRunFinalized):
			writeErr(w, err, http.StatusConflict)
		case errors.Is(err, routine.ErrRunNotFound):
			writeErr(w, err, http.StatusNotFound)
		default:
			writeErr(w, err, http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// cancelRoutine aborts a routine run and returns the updated run. A terminal run
// (done or cancelled) → 409; an unknown run → 404; any other error → 500.
func (s *Server) cancelRoutine(w http.ResponseWriter, r *http.Request) {
	run, err := s.routines.Cancel(r.Context(), r.PathValue("id"))
	if err != nil {
		switch {
		case errors.Is(err, routine.ErrRunNotCancelable):
			writeErr(w, err, http.StatusConflict)
		case errors.Is(err, routine.ErrRunNotFound):
			writeErr(w, err, http.StatusNotFound)
		default:
			writeErr(w, err, http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// --- response DTOs ---

type previewTagResp struct {
	NextTag   string `json:"nextTag"`
	LastTag   string `json:"lastTag"`
	FeatCount int    `json:"featCount"`
	FixCount  int    `json:"fixCount"`
}

type preflightCheckResp struct {
	Capability string `json:"capability"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
}

type preflightResp struct {
	TokenScopes     []string             `json:"tokenScopes"`
	ScopesKnown     bool                 `json:"scopesKnown"`
	AccessLevel     int                  `json:"accessLevel"`
	AccessLevelName string               `json:"accessLevelName"`
	DefaultBranch   string               `json:"defaultBranch"`
	Checks          []preflightCheckResp `json:"checks"`
}

func toPreflight(pf routine.Preflight) preflightResp {
	scopes := pf.TokenScopes
	if scopes == nil {
		scopes = []string{}
	}
	checks := make([]preflightCheckResp, 0, len(pf.Checks))
	for _, c := range pf.Checks {
		checks = append(checks, preflightCheckResp{
			Capability: string(c.Capability), Label: c.Label, Status: string(c.Status), Detail: c.Detail,
		})
	}
	return preflightResp{
		TokenScopes:     scopes,
		ScopesKnown:     pf.ScopesKnown,
		AccessLevel:     pf.AccessLevel,
		AccessLevelName: pf.AccessLevelName,
		DefaultBranch:   pf.DefaultBranch,
		Checks:          checks,
	}
}

type routineStepResp struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type routineRunResp struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	RepoID    string            `json:"repoId"`
	MRIID     int               `json:"mrIid"`
	Status    string            `json:"status"`
	Steps     []routineStepResp `json:"steps"`
	State     json.RawMessage   `json:"state"`
	LastError string            `json:"lastError"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Archived  bool              `json:"archived"`
	// RepoName is a best-effort repo display name, populated only on the global
	// recent-runs list (listRecentRoutines); it stays empty (and omitted) on the
	// per-repo and single-run paths, which already know their repo context.
	RepoName string `json:"repoName,omitempty"`
	// Flow, SourceBranch and TargetBranch are decoded best-effort from a release
	// run's immutable params so the UI can show which branches a run moves and
	// under which flow ("development" or "main"). They are omitted for non-release
	// runs and when the params cannot be decoded.
	Flow         string `json:"flow,omitempty"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
}

// releaseParamsView is the subset of a release run's persisted params the run
// DTO surfaces. It is decoded best-effort from run.Params (JSON), so a decode
// error simply leaves the branch fields empty rather than failing the response.
type releaseParamsView struct {
	Flow         string `json:"flow"`
	SourceBranch string `json:"sourceBranch"`
	TargetBranch string `json:"targetBranch"`
}

func toRun(run routine.Run) routineRunResp {
	steps := make([]routineStepResp, 0, len(run.Steps))
	for _, st := range run.Steps {
		steps = append(steps, routineStepResp{
			Name: st.Name, Status: string(st.Status), Detail: st.Detail, UpdatedAt: st.UpdatedAt,
		})
	}
	state := run.State
	if len(state) == 0 {
		state = json.RawMessage("{}")
	}
	resp := routineRunResp{
		ID:        run.ID,
		Kind:      string(run.Kind),
		RepoID:    run.RepoID,
		MRIID:     run.MRIID,
		Status:    string(run.Status),
		Steps:     steps,
		State:     state,
		LastError: run.LastError,
		CreatedAt: run.CreatedAt,
		UpdatedAt: run.UpdatedAt,
		Archived:  run.Archived,
	}
	// Best-effort: expose a release run's flow and branches from its immutable
	// params (persisted for every release run, so existing runs work too). A
	// decode error just leaves the fields empty.
	if run.Kind == routine.KindRelease && len(run.Params) > 0 {
		var pv releaseParamsView
		if err := json.Unmarshal(run.Params, &pv); err == nil {
			resp.Flow = pv.Flow
			resp.SourceBranch = pv.SourceBranch
			resp.TargetBranch = pv.TargetBranch
		}
	}
	return resp
}
