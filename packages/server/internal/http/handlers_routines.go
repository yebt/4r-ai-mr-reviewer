package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
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
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// listRoutines returns a repo's routine runs, newest first.
func (s *Server) listRoutines(w http.ResponseWriter, r *http.Request) {
	runs, err := s.routines.ListByRepo(r.Context(), r.PathValue("id"))
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

// resumeRoutine re-queues a blocked run. A non-blocked run → 409.
func (s *Server) resumeRoutine(w http.ResponseWriter, r *http.Request) {
	run, err := s.routines.Resume(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, routine.ErrNotResumable) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toRun(run))
}

// --- response DTOs ---

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
	return routineRunResp{
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
	}
}
