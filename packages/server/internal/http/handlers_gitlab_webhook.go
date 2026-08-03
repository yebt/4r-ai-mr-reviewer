package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"time"

	glapi "github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
)

// gitlabWebhook receives GitLab merge-request events for one repo and, when the
// repo has the webhook enabled and the event warrants it, auto-creates and runs
// a review for the merge request.
//
// The endpoint is per-repo (the repo id is in the path) and self-secured: it
// validates the X-Gitlab-Token header against the repo's stored secret in
// constant time, so the auth middleware exempts it (see requireAuth). A disabled
// repo answers 200 without doing anything, so a lingering GitLab webhook goes
// quiet instead of erroring.
//
// Dispatch runs on a detached goroutine with its own bounded context and a panic
// recovery: GitLab expects a fast 2xx and retries on failure, so the handler must
// never block on review creation nor let it crash the request.
func (s *Server) gitlabWebhook(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("id")

	rp, err := s.repos.Get(r.Context(), repoID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "repo not found"})
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}

	// Disabled: the receiver is inert. Answer 200 so GitLab stops retrying.
	if !rp.WebhookEnabled {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "webhook disabled"})
		return
	}

	// Validate the secret in constant time. An empty stored secret never accepts
	// a token (a blank X-Gitlab-Token must not authenticate an enabled repo).
	got := r.Header.Get("X-Gitlab-Token")
	if rp.WebhookSecret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(rp.WebhookSecret)) != 1 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var ev glapi.MergeRequestEvent
	if err := decode(r, &ev); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	// Ignore non-MR events (push, pipeline, ...).
	if ev.ObjectKind != "merge_request" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "not a merge_request event"})
		return
	}

	// Trigger on open/reopen, and on an update that carries new commits (oldrev
	// set = a push to the source branch). Ignore merge/close/approve and
	// metadata-only updates so we only review actual code changes.
	action := ev.ObjectAttributes.Action
	trigger := action == "open" || action == "reopen" ||
		(action == "update" && ev.ObjectAttributes.OldRev != "")
	if !trigger {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "action " + action + " does not trigger a review"})
		return
	}

	iid := ev.ObjectAttributes.IID

	// Dispatch off the request path so GitLab gets a fast 200. A duplicate
	// (pending/running) review is skipped inside TriggerFromWebhook.
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("gitlab webhook: dispatch panic: %v", rec)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rv, created, err := s.reviews.TriggerFromWebhook(ctx, repoID, iid)
		switch {
		case err != nil:
			log.Printf("gitlab webhook: trigger review for repo %s MR !%d: %v", repoID, iid, err)
		case !created:
			log.Printf("gitlab webhook: skipped review for repo %s MR !%d (already active)", repoID, iid)
		default:
			log.Printf("gitlab webhook: created review %s for repo %s MR !%d", rv.ID, repoID, iid)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}
