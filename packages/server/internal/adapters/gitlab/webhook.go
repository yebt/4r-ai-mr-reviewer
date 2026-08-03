package gitlab

// MergeRequestEvent is the subset of a GitLab "Merge request" webhook event the
// reviewer needs to decide whether to auto-trigger a review. GitLab sends many
// fields; only these are decoded.
//
// See https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html#merge-request-events
type MergeRequestEvent struct {
	// ObjectKind is "merge_request" for MR events. Other event kinds (push,
	// pipeline, ...) carry a different value and are ignored.
	ObjectKind string `json:"object_kind"`

	// ObjectAttributes carries the MR's current state and the action that
	// produced the event.
	ObjectAttributes struct {
		IID    int    `json:"iid"`
		Action string `json:"action"` // open, reopen, update, close, merge, approved, ...
		State  string `json:"state"`  // opened, closed, merged, locked
		// OldRev is set only on an "update" event that carries new commits (a
		// push to the source branch); it is empty for metadata-only updates
		// (label/assignee/description changes).
		OldRev       string `json:"oldrev"`
		TargetBranch string `json:"target_branch"`
		SourceBranch string `json:"source_branch"`
	} `json:"object_attributes"`

	// Project identifies the GitLab project the MR belongs to.
	Project struct {
		ID                int    `json:"id"`
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
}
