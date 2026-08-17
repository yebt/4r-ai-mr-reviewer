package httpapi

import (
	"errors"
	"net/http"

	"github.com/webcloster-dev/ai-reviewer/internal/app/mergerequests"
)

// generateMRResp is the AI-drafted title+description returned before the merge
// request is opened, so the client can show it in editable fields.
type generateMRResp struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// generateMergeRequest drafts a title+description for a merge request from the
// diff between two branches, optionally in a profile's author voice. It does
// not open the merge request.
func (s *Server) generateMergeRequest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SourceBranch string `json:"sourceBranch"`
		TargetBranch string `json:"targetBranch"`
		ProfileID    string `json:"profileId"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	gen, err := s.mergeRequests.Generate(r.Context(), r.PathValue("id"), in.SourceBranch, in.TargetBranch, in.ProfileID)
	if err != nil {
		writeMergeRequestErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, generateMRResp{Title: gen.Title, Description: gen.Description})
}

// createMergeRequest opens a merge request with the (possibly edited) title and
// description the user reviewed after generation.
func (s *Server) createMergeRequest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SourceBranch string `json:"sourceBranch"`
		TargetBranch string `json:"targetBranch"`
		Title        string `json:"title"`
		Description  string `json:"description"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	mr, err := s.mergeRequests.Create(r.Context(), r.PathValue("id"), in.SourceBranch, in.TargetBranch, in.Title, in.Description)
	if err != nil {
		writeMergeRequestErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMR(mr))
}

// writeMergeRequestErr maps the service's sentinels to status codes: a bad
// branch selection is 400, an unready profile style guide is 409, and anything
// else (an unknown repo → 404 via writeErr, or an upstream GitLab/AI failure)
// falls through to a 400 that surfaces the underlying message to the form.
func writeMergeRequestErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, mergerequests.ErrSameBranch), errors.Is(err, mergerequests.ErrNoChanges):
		writeErr(w, err, http.StatusBadRequest)
	case errors.Is(err, mergerequests.ErrStyleGuideNotReady):
		writeErr(w, err, http.StatusConflict)
	default:
		writeErr(w, err, http.StatusBadRequest)
	}
}
